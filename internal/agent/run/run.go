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
	latMemory        *reviewMemory
	latObserved      bool
	latReviewed      bool
	reviewing        bool
	latSteering      []string
	reviewGenerator  modelapi.Generator
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
	commandRevision  uint64
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
			if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
				return ctx.Err()
			}
			return errors.Join(errRuntimeRecord, err)
		}
		return nil
	}
	if support, ok := r.Generator.(interface{ ImageInputSupported() bool }); ok {
		r.ImageInput = r.ImageInput && support.ImageInputSupported()
	}
	reviewGenerator := r.Generator
	if scoped, ok := r.Generator.(interface {
		NewRunGenerator() (modelapi.Generator, func())
	}); ok {
		generator, release := scoped.NewRunGenerator()
		defer release()
		r.Generator = generator
	}
	s := &execution{runner: r, limits: limits, system: system, emit: emit, definitions: imageDefinitions(r.Tools.Definitions(), r.ImageInput), statistics: runStatistics{started: started}}
	if r.LAT {
		s.reviewGenerator = reviewGenerator
	}
	s.usage = &requestBudget{}
	s.usage.begin(limits)
	if r.LAT {
		owner, stop := context.WithDeadline(ctx, s.usage.snapshot().deadline)
		s.latMemory = &reviewMemory{ctx: owner, stop: stop, dirty: true}
		s.latMemory.controller.Repository, s.latMemory.controller.Session = r.ReviewMemory, r.Session
		s.latMemory.controller.Redact = r.Redact
	}

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
		err = errors.Join(err, s.closeLATMemory(ctx))
		err = errors.Join(err, s.stopCommands(ctx))
		err = s.settle(ctx, err)
		result = s.result
	}()
	if commands, ok := r.Tools.(toolapi.CommandExecutor); ok {
		commands.BeginCommands(ctx, s.usage.snapshot().deadline)
	}
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
		if _, err = s.collectCommands(ctx); err != nil {
			return s.result, err
		}
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
		commandRevision := s.commandRevision
		reply, askErr := s.askResponse(ctx)
		if askErr == nil {
			s.capacityRecovery = false
		}
		if askErr != nil {
			if handled, imageErr := s.recoverImages(ctx, askErr); handled {
				if imageErr != nil {
					return s.result, imageErr
				}
				continue
			}
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
			if commandRevision != s.commandRevision {
				continue
			}
			if pending, pendingErr := s.pendingCommands(ctx); pendingErr != nil {
				return s.result, pendingErr
			} else if pending {
				continue
			}
			if continued, reviewErr := s.reviewWork(ctx, reply); reviewErr != nil {
				return s.result, reviewErr
			} else if continued {
				continue
			}
			return s.result, s.finish(ctx, reply)
		}
	}
}
