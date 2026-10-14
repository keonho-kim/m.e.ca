// Purpose: Run a bounded Ralph task with full uncompacted context, sequential tool dialogue and evidence-gated completion.
// Pattern: State Machine / Use Case.
// Usage: Invoked for one user objective or approved Plan step; cancellation never replays effects.
// Related: internal/agent/run/action.go, internal/agent/run/generate.go, internal/agent/run/events.go.
package run

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/pfc"
	planref "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/progress"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/agent/verification"
	"time"
)

type task struct {
	files           progress.Files
	runner          Runner
	limits          Limits
	system          string
	planning        bool
	result          Result
	checkpoint      Checkpoint
	emit            func(Event) error
	definitions     []toolapi.Definition
	seen            map[[32]byte]bool
	mutations       map[[32]byte]bool
	worked, changed bool
	usage           *Usage
	messages        []modelapi.Message
	memoryContext   string
}

func (r Runner) Run(ctx context.Context, input string, planning bool, approve Approver, emit func(Event) error) (result Result, err error) {
	if r.Generator == nil || r.Tools == nil || r.Journal == nil || r.Redact == nil || approve == nil || emit == nil {
		return result, fmt.Errorf("task dependencies are required")
	}
	if r.PreviousTask != nil {
		if err := r.PreviousTask.Check(); err != nil {
			return result, err
		}
	}
	if err = r.Settings.Check(); err != nil {
		return result, err
	}
	if r.Inference != nil {
		if err := r.Inference.Check(); err != nil {
			return result, err
		}
	}
	limits := r.Limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	if err = limits.Check(); err != nil {
		return result, err
	}
	templates := r.templates()
	base := templates.System
	system, err := prompt.Compose(base, r.ProjectInstructions, planning)
	if err != nil {
		return result, err
	}
	if len(input) > 64*1024 {
		return result, fmt.Errorf("objective exceeds 64 KiB")
	}
	if r.observer == nil {
		observer, err := r.startMonitor(ctx)
		if err != nil {
			return result, err
		}
		defer observer.Close()
		r.observer = observer
	}
	s := &task{runner: r, limits: limits, system: system, planning: planning, emit: emit, definitions: r.Tools.Definitions(), seen: map[[32]byte]bool{}, mutations: map[[32]byte]bool{}}
	s.usage = r.Usage
	if s.usage == nil {
		s.usage = &Usage{}
	}
	if s.usage.deadline.IsZero() {
		s.usage.deadline = time.Now().Add(time.Duration(limits.MaxSeconds) * time.Second)
	}

	digest := sha256.Sum256([]byte(r.ProjectInstructions))
	s.checkpoint = Checkpoint{ID: "task_" + rand.Text(), Objective: r.Redact(input), Status: Running, Phase: "decision", Step: r.Step, InstructionsDigest: fmt.Sprintf("%x", digest)}
	if r.assignment != nil {
		s.checkpoint.GoalID = r.assignment.GoalID
		s.checkpoint.ID = r.assignment.TaskID
	}
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
	if !r.internalTask {
		if err = s.save(ctx, modelapi.Message{Role: "user", Content: input}); err != nil {
			return s.result, err
		}
	}
	for s.usage.Iterations < limits.MaxIterations {
		if err = s.checkTimeBudget(ctx); err != nil {
			return s.result, err
		}
		s.usage.Iterations++
		s.checkpoint.Iteration = s.usage.Iterations
		s.checkpoint.ModelCalls = s.usage.ModelCalls
		s.checkpoint.Phase = "decision"
		s.checkpoint.Tool = ""
		s.checkpoint.Parameter = ""
		s.checkpoint.Arguments = nil
		s.checkpoint.Checking = false
		if err = s.persist(ctx); err != nil {
			return s.result, err
		}
		if err = s.refreshMemory(ctx); err != nil {
			return s.result, err
		}
		role := inference.Agent
		if planning {
			role = inference.Plan
		}
		text, askErr := s.ask(ctx, "DECISION: Choose one next action. For a greeting or a question answerable without project evidence, use DONE followed by the answer in any mode; tools are optional and must serve the user goal. In Plan mode, ordinary discussion does not require a completed Plan. To enter PFC, return only {\"tool\":\"tool_name\"} using the provided tool list, without arguments or other keys. A completion claim alone is not verification.", role, 16384)
		if askErr != nil {
			return s.result, askErr
		}
		decision, parseErr := pfc.ParseDecision(text)
		if parseErr != nil {
			s.checkpoint.Observation = "Invalid decision: " + parseErr.Error()
			s.checkpoint.NoProgress++
		} else {
			switch decision.Kind {
			case pfc.Done:
				if planning {
					s.checkpoint.Status = Answered
					if _, err := planref.ParsePlanProposal(decision.Text); err == nil {
						s.checkpoint.Status = PlanReady
					}
				} else if s.checkpoint.PendingReview || !s.worked && r.assignment != nil && r.assignment.Kind == "work" {
					s.checkpoint.Observation = "DONE rejected: this work Task still needs execution and verification; a rejected Conclave candidate cannot be treated as completed."
					s.checkpoint.NoProgress++
					break
				} else if !s.worked {
					s.checkpoint.Status = Answered
				} else if s.checkpoint.Evidence != "" && len(verification.Pending(s.checkpoint.Checks, s.checkpoint.ChangeRevision)) == 0 {
					s.checkpoint.Status = Verified
				} else {
					s.checkpoint.Observation = "DONE rejected: satisfy every pending or stale verification in Current task facts after the latest change. Read-back alone does not verify a change; use an approved check command or WAIT for review."
					s.checkpoint.NoProgress++
					break
				}
				s.result.Text = decision.Text
				if err = s.save(ctx, modelapi.Message{Role: "assistant", Content: decision.Text}); err != nil {
					return s.result, err
				}
				err = s.emit(Event{Kind: "response", Text: decision.Text})
				return s.result, err
			case pfc.Wait:
				s.checkpoint.Status = NeedsInput
				s.result.Text = decision.Text
				if err = s.save(ctx, modelapi.Message{Role: "assistant", Content: decision.Text}); err != nil {
					return s.result, err
				}
				err = s.emit(Event{Kind: "response", Text: decision.Text})
				return s.result, err
			case pfc.Continue:
				s.checkpoint.Observation = "No tool observation was produced. Next intention: " + excerpt(decision.Text, 1024)
				s.checkpoint.NoProgress++
			case pfc.Tool:
				if err = s.action(ctx, decision, approve); err != nil {
					return s.result, err
				}
			}
		}
		if s.checkpoint.NoProgress >= limits.StallIterations {
			if err := s.recoverProgress(ctx); err != nil {
				return s.result, err
			}
		}
	}
	return s.result, fmt.Errorf("iteration budget exhausted after %d actions", limits.MaxIterations)
}
