// Purpose: Run a bounded native-tool conversation with full uncompacted context, native tool calls and persisted decision state.
// Pattern: State Machine / Use Case.
// Usage: Invoked for one user objective or user request; cancellation never replays effects.
// Related: internal/agent/run/action.go, internal/agent/run/generate.go, internal/agent/record/events.go.
package run

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	contextbuild "github.com/keonho-kim/meca/internal/agent/context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"time"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type execution struct {
	projection       contextbuild.Projection
	historyLoaded    bool
	runner           Runner
	limits           Limits
	system           string
	result           Result
	checkpoint       runrecord.Checkpoint
	emit             func(runrecord.Event) error
	definitions      []toolapi.Definition
	usage            *requestBudget
	statistics       runStatistics
	messages         []modelapi.Message
	memoryContext    string
	capacityRecovery bool
}

func (r Runner) Run(ctx context.Context, input string, approve Approver, emit func(runrecord.Event) error) (result Result, err error) {
	started := time.Now()
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
	templates := prompt.Defaults()
	if r.Prompts != nil {
		templates = *r.Prompts
	}
	system, err := prompt.Work(templates.System, templates.Development, r.ProjectInstructions)
	if err != nil {
		return result, err
	}
	if len(input) > 64*1024*1024 {
		return result, fmt.Errorf("objective exceeds 64 MiB")
	}
	originalEmit := emit
	emit = func(event runrecord.Event) error {
		if err := originalEmit(event); err != nil {
			return errors.Join(errRuntimeRecord, err)
		}
		return nil
	}
	s := &execution{runner: r, limits: limits, system: system, emit: emit, definitions: r.Tools.Definitions(), statistics: runStatistics{started: started}}
	s.usage = &requestBudget{}
	s.usage.begin(limits)

	digest := sha256.Sum256([]byte(r.ProjectInstructions))
	id := "run_" + rand.Text()
	s.checkpoint = runrecord.Checkpoint{ID: id, Objective: r.Redact(input), Status: runrecord.Running, Phase: "decision", InstructionsDigest: fmt.Sprintf("%x", digest)}
	if r.PreviousRun != nil {
		for _, file := range r.PreviousRun.Work.Files {
			file.Changed = false
			s.checkpoint.Work.Files = append(s.checkpoint.Work.Files, file)
		}
	}
	defer func() {
		err = s.settle(ctx, err)
		result = s.result
	}()
	if previous := r.PreviousRun; previous != nil && (previous.Status == runrecord.Interrupted || previous.Status == runrecord.Stopped) {
		note := "Previous request: " + previous.Objective + "\nLast observed outcome: " + previous.Observation
		if previous.Blocker != "" {
			note += "\nObserved blocker: " + previous.Blocker
		}
		if err = s.save(ctx, modelapi.Message{Role: "system", Content: note}); err != nil {
			return s.result, err
		}
	}
	if err = s.save(ctx, modelapi.Message{Role: "user", Content: input}); err != nil {
		return s.result, err
	}
	for {
		if _, err = s.steer(ctx); err != nil {
			return s.result, err
		}
		if finished, decisionErr := s.prepareDecision(ctx); finished || decisionErr != nil {
			return s.result, decisionErr
		}
		s.checkpoint.Iteration = s.usage.snapshot().ToolCalls
		s.checkpoint.ModelCalls = s.usage.snapshot().ModelCalls
		s.checkpoint.Phase = "decision"
		s.checkpoint.Tool = ""
		s.checkpoint.Arguments = nil
		if err = s.persist(ctx); err != nil {
			return s.result, err
		}
		if err = s.refreshMemory(ctx, false); err != nil {
			return s.result, err
		}
		// Compaction can take time; admit instructions received during it before
		// generating another answer or proposing effects from the earlier request.
		if changed, steerErr := s.steer(ctx); steerErr != nil {
			return s.result, steerErr
		} else if changed {
			if err = s.persist(ctx); err != nil {
				return s.result, err
			}
		}
		reply, askErr := s.askResponse(ctx)
		if askErr == nil {
			s.capacityRecovery = false
		}
		if askErr != nil {
			var capacity *modelapi.ContextLimitError
			if !s.capacityRecovery && errors.As(askErr, &capacity) {
				s.capacityRecovery = true
				if m := s.runner.Memory; m != nil && capacity.Capacity > 0 && (m.ContextWindow == 0 || capacity.Capacity < m.ContextWindow) {
					m.ContextWindow = capacity.Capacity
				}
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
			s.checkpoint.Observation = "Model response failed before execution: " + s.runner.Redact(askErr.Error())
			if err := s.save(ctx, modelapi.Message{Role: "system", Content: s.checkpoint.Observation}); err != nil {
				return s.result, err
			}
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
			if queued, err := s.continueSteered(ctx, reply); err != nil {
				return s.result, err
			} else if queued {
				continue
			}
			return s.result, s.finish(ctx, reply)
		}
	}
}
