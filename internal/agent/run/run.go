// Purpose: Run a bounded Ralph task with fresh context, sequential tool dialogue and evidence-gated completion.
// Pattern: State Machine / Use Case.
// Usage: Invoked for one user objective or approved Plan step; cancellation never replays effects.
// Related: internal/agent/run/action.go, internal/agent/run/generate.go, internal/agent/run/types.go.
package run

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/dialogue"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"time"
)

type task struct {
	runner          Runner
	limits          Limits
	system          string
	planning        bool
	result          Result
	checkpoint      Checkpoint
	emit            func(Event) error
	definitions     []agent.ToolDefinition
	seen            map[[32]byte]bool
	mutations       map[[32]byte]bool
	worked, changed bool
	usage           *Usage
	background      []agent.Message
	memoryContext   string
}

func (r Runner) Run(ctx context.Context, history []agent.Message, input string, planning bool, approve Approver, emit func(Event) error) (result Result, err error) {
	if r.Generator == nil || r.Tools == nil || r.Journal == nil || r.Redact == nil || approve == nil || emit == nil {
		return result, fmt.Errorf("task dependencies are required")
	}
	if err = r.Settings.Check(); err != nil {
		return result, err
	}
	limits := r.Limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	if err = limits.Check(); err != nil {
		return result, err
	}
	base := r.BasePrompt
	if base == "" {
		base = prompt.Defaults().System
	}
	system, err := prompt.Compose(base, r.ProjectInstructions, planning)
	if err != nil {
		return result, err
	}
	if len(input) > 64*1024 {
		return result, fmt.Errorf("objective exceeds 64 KiB")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(limits.MaxSeconds)*time.Second)
	defer cancel()
	s := &task{runner: r, limits: limits, system: system, planning: planning, emit: emit, definitions: r.Tools.Definitions(), seen: map[[32]byte]bool{}, mutations: map[[32]byte]bool{}}
	s.usage = r.Usage
	if s.usage == nil {
		s.usage = &Usage{}
	}
	s.background = background(history)
	s.result.Messages = append([]agent.Message(nil), history...)
	digest := sha256.Sum256([]byte(r.ProjectInstructions))
	s.checkpoint = Checkpoint{ID: "task_" + rand.Text(), Objective: r.Redact(input), Status: Running, Phase: "decision", Step: r.Step, InstructionsDigest: fmt.Sprintf("%x", digest)}
	defer func() {
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				s.checkpoint.Status = Interrupted
			} else {
				s.checkpoint.Status = Stopped
			}
		}
		if err == nil {
			s.checkpoint.Phase = "finished"
		}
		s.result.Status = s.checkpoint.Status
		s.result.Evidence = s.checkpoint.Evidence
		flushCtx, flushCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer flushCancel()
		err = errors.Join(err, s.persist(flushCtx))
		result = s.result
	}()
	if err = s.save(ctx, agent.Message{Role: "user", Content: input}); err != nil {
		return s.result, err
	}
	for s.usage.Iterations < limits.MaxIterations {
		s.usage.Iterations++
		s.checkpoint.Iteration = s.usage.Iterations
		s.checkpoint.ModelCalls = s.usage.ModelCalls
		s.checkpoint.Phase = "decision"
		s.checkpoint.Tool = ""
		s.checkpoint.Parameter = ""
		s.checkpoint.Arguments = nil
		if err = s.persist(ctx); err != nil {
			return s.result, err
		}
		if err = s.refreshMemory(ctx); err != nil {
			return s.result, err
		}
		text, askErr := s.ask(ctx, "DECISION: Choose one next action. Outside Plan mode, for a greeting or a question answerable without project evidence, use DONE followed by the answer; tools are optional and must serve the user goal. Use TOOL/CHECK only when necessary. A completion claim alone is not verification.", 2048, 16384)
		if askErr != nil {
			return s.result, askErr
		}
		decision, parseErr := dialogue.ParseDecision(text)
		if parseErr != nil {
			s.checkpoint.Observation = "Invalid decision: " + parseErr.Error()
			s.checkpoint.NoProgress++
		} else {
			switch decision.Kind {
			case dialogue.Done:
				if planning {
					s.checkpoint.Status = PlanReady
				} else if !s.worked {
					s.checkpoint.Status = Answered
				} else if s.checkpoint.Evidence != "" {
					s.checkpoint.Status = Verified
				} else {
					s.checkpoint.Observation = "DONE rejected: perform a successful CHECK after the latest change. Read-back alone does not verify a change; use an approved check command or WAIT for review."
					s.checkpoint.NoProgress++
					break
				}
				s.result.Text = decision.Text
				if err = s.save(ctx, agent.Message{Role: "assistant", Content: decision.Text}); err != nil {
					return s.result, err
				}
				err = s.emit(Event{Kind: "response", Text: decision.Text})
				return s.result, err
			case dialogue.Wait:
				s.checkpoint.Status = NeedsInput
				s.result.Text = decision.Text
				if err = s.save(ctx, agent.Message{Role: "assistant", Content: decision.Text}); err != nil {
					return s.result, err
				}
				err = s.emit(Event{Kind: "response", Text: decision.Text})
				return s.result, err
			case dialogue.Continue:
				s.checkpoint.Observation = "No tool observation was produced. Next intention: " + excerpt(decision.Text, 1024)
				s.checkpoint.NoProgress++
			case dialogue.Tool, dialogue.Check:
				if err = s.action(ctx, decision, approve); err != nil {
					return s.result, err
				}
			}
		}
		if s.checkpoint.NoProgress >= limits.StallIterations {
			return s.result, fmt.Errorf("task stalled after %d iterations without new tool evidence", limits.StallIterations)
		}
	}
	return s.result, fmt.Errorf("iteration budget exhausted after %d actions", limits.MaxIterations)
}
