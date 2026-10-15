// Purpose: Run a bounded Ralph execution with full uncompacted context, native tool calls and persisted decision state.
// Pattern: State Machine / Use Case.
// Usage: Invoked for one user objective or user request; cancellation never replays effects.
// Related: internal/agent/run/action.go, internal/agent/run/generate.go, internal/agent/run/events.go.
package run

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/progress"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"time"
)

type execution struct {
	textCorrection   bool
	terminal         bool
	files            progress.Files
	runner           Runner
	limits           Limits
	system           string
	result           Result
	checkpoint       Checkpoint
	emit             func(Event) error
	definitions      []toolapi.Definition
	seen             map[[32]byte]bool
	mutations        map[[32]byte]bool
	commands         map[[32]byte]int
	usage            *Usage
	messages         []modelapi.Message
	memoryContext    string
	capacityRecovery bool
	reviews          map[[32]byte]conclave.Result
	evidenceRevision int
}

func (r Runner) Run(ctx context.Context, input string, approve Approver, emit func(Event) error) (result Result, err error) {
	if r.Generator == nil || r.Tools == nil || r.Journal == nil || r.Redact == nil || approve == nil || emit == nil {
		return result, fmt.Errorf("execution dependencies are required")
	}
	if r.PreviousRun != nil {
		if err := r.PreviousRun.Check(); err != nil {
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
	system, err := prompt.Compose(base, r.ProjectInstructions)
	if err != nil {
		return result, err
	}
	if len(input) > 16*1024*1024 {
		return result, fmt.Errorf("objective exceeds 16 MiB")
	}
	if r.observer == nil {
		observer, err := r.startMonitor(ctx)
		if err != nil {
			return result, err
		}
		defer observer.Close()
		r.observer = observer
	}
	originalEmit := emit
	emit = func(event Event) error {
		if err := originalEmit(event); err != nil {
			return errors.Join(errRuntimeRecord, err)
		}
		return nil
	}
	s := &execution{runner: r, limits: limits, system: system, emit: emit, definitions: r.Tools.Definitions(), seen: map[[32]byte]bool{}, mutations: map[[32]byte]bool{}, commands: map[[32]byte]int{}}
	s.usage = r.Usage
	if s.usage == nil {
		s.usage = &Usage{}
	}
	if s.usage.deadline.IsZero() {
		s.usage.deadline = time.Now().Add(time.Duration(limits.MaxSeconds) * time.Second)
	}

	digest := sha256.Sum256([]byte(r.ProjectInstructions))
	s.checkpoint = Checkpoint{ID: "run_" + rand.Text(), Objective: r.Redact(input), Status: Running, Phase: "decision", InstructionsDigest: fmt.Sprintf("%x", digest)}
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
		flushCtx, flushCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer flushCancel()
		err = errors.Join(err, s.persist(flushCtx))
		result = s.result
	}()
	if err = s.save(ctx, modelapi.Message{Role: "user", Content: input}); err != nil {
		return s.result, err
	}
	for s.usage.Iterations < limits.MaxIterations {
		if err = s.checkTimeBudget(ctx); err != nil {
			if ctx.Err() != nil {
				return s.result, ctx.Err()
			}
			return s.report(ctx, "execution time budget exhausted", true)
		}
		if s.usage.ModelCalls >= s.limits.MaxModelCalls {
			return s.report(ctx, "model call budget reserved for final reporting", true)
		}
		s.usage.Iterations++
		s.checkpoint.Iteration = s.usage.Iterations
		s.checkpoint.ModelCalls = s.usage.ModelCalls
		s.checkpoint.Phase = "decision"
		s.checkpoint.Tool = ""
		s.checkpoint.Arguments = nil
		if err = s.persist(ctx); err != nil {
			return s.result, err
		}
		if err = s.refreshMemory(ctx, false); err != nil {
			return s.result, err
		}
		role := inference.Agent

		reply, askErr := s.askResponse(ctx, "Continue the requested work using the available results, or answer with the outcome. Explain any remaining work honestly.", role, modelapi.MaxResponseBytes)
		if askErr != nil {
			var capacity interface{ ContextLimit() bool }
			if !s.capacityRecovery && errors.As(askErr, &capacity) && capacity.ContextLimit() {
				s.capacityRecovery = true
				if err = s.refreshMemory(ctx, true); err != nil {
					return s.result, errors.Join(askErr, err)
				}
				continue
			}
			var malformed *modelapi.ToolCallError
			var truncated *modelapi.OutputLimitError
			if !errors.As(askErr, &malformed) && !errors.As(askErr, &truncated) {
				return s.result, askErr
			}
			s.result.HadToolFailure = true
			s.checkpoint.Observation = "Correct the native tool call or narrow the change to fit the output limit; nothing executed: " + s.runner.Redact(askErr.Error())
			s.checkpoint.NoProgress++
		} else if len(reply.Calls) > 0 {
			if len(reply.Calls) > 1 {
				err = s.batch(ctx, reply, approve)
			} else {
				err = s.action(ctx, reply, approve)
			}
			if err != nil {
				return s.result, err
			}

		} else {
			finished, finishErr := s.finishOrReview(ctx, reply.Text)
			if finishErr != nil {
				return s.result, finishErr
			}
			if finished {
				return s.result, nil
			}
		}

		if s.terminal {
			return s.result, nil
		}
		if s.checkpoint.ReportReason != "" {
			return s.report(ctx, s.checkpoint.ReportReason, true)
		}
		if s.checkpoint.NoProgress >= limits.StallIterations {
			if err := s.recoverProgress(ctx); err != nil {
				return s.result, err
			}
			if s.terminal {
				return s.result, nil
			}
			if s.checkpoint.ReportReason != "" {
				return s.report(ctx, s.checkpoint.ReportReason, true)
			}
		}
	}
	return s.report(ctx, fmt.Sprintf("iteration budget exhausted after %d actions", limits.MaxIterations), true)
}
