// Purpose: Schedule validated native calls in bounded independent waves and preserve paired results.
// Pattern: Serial owner with bounded parallel execution.
// Usage: Run dispatches multi-call responses here; one owner mutates runtime and journal state.
// Related: internal/agent/run/batch_wave.go, internal/agent/run/call_validation.go.
package run

import (
	"context"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/policy"

	// An executor opts in only when it can resolve aliases and reason about actual accesses.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const maxParallelTools = 16

type parallelExecutor interface {
	ParallelBatch(context.Context, policy.Settings, []toolapi.Call, int) (toolapi.BatchPlan, error)
}

func (s *execution) batch(ctx context.Context, reply modelapi.Response, approve Approver) error {
	s.usage.recordTools(len(reply.Calls))
	calls := make([]validatedCall, len(reply.Calls))
	for i, call := range reply.Calls {

		value, err := s.validateCall(call)
		if err != nil {
			return s.returnToolFeedback(ctx, reply, "Batch validation failed; nothing executed: "+err.Error())
		}
		calls[i] = value
	}
	s.checkpoint.Iteration = s.usage.ToolCalls
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, ReasoningState: reply.ReasoningState, Calls: reply.Calls}); err != nil {
		return err
	}
	if reply.Text != "" {
		if err := s.emit(runrecord.Event{Kind: "progress", Text: reply.Text}); err != nil {
			return err
		}
	}
	instructionRevision := s.instructionRevision
	for start := 0; start < len(calls); {
		if s.result.Departed {
			return s.skipCalls(ctx, calls[start:], skipForClosure, "The session has closed; these calls did not execute.")
		}
		if err := s.checkSteering(ctx, instructionRevision); err != nil {
			var failure *faults.Failure
			if errors.As(err, &failure) && failure.Code == diagcatalog.InstructionChanged {
				return s.skipCalls(ctx, calls[start:], skipForInstruction, failure.Detail)
			}
			return errors.Join(err, s.skipCalls(ctx, calls[start:], skipForContext, "Instruction admission failed before this wave; nothing in it executed."))
		}
		plan, err := s.planWave(ctx, calls, start)
		if err != nil {
			return errors.Join(s.skipCalls(ctx, calls[start:], skipAfterFailure, err.Error()), fatalBatchError(ctx, err))
		}
		end := start + plan.Count
		failed, err := s.batchWave(ctx, calls[start:end], plan.PreparationConcurrency, approve)
		if failed || err != nil {
			reason := skipAfterFailure
			if ctx.Err() != nil {
				reason = skipForInterruption
			} else if instructionRevision != s.instructionRevision {
				reason = skipForInstruction
			}
			skipErr := s.skipCalls(ctx, calls[end:], reason, "Earlier batch operation failed or was cancelled; inspect recorded outcomes before considering remaining work.")
			return errors.Join(err, skipErr)
		}
		start = end
	}
	return nil
}

func (s *execution) planWave(ctx context.Context, calls []validatedCall, start int) (toolapi.BatchPlan, error) {
	planner, ok := s.runner.Tools.(parallelExecutor)
	if !ok {
		return toolapi.BatchPlan{Count: 1, PreparationConcurrency: 1}, nil
	}
	var batch [maxParallelTools]toolapi.Call
	count := min(len(calls)-start, len(batch))
	for i := range count {
		batch[i] = calls[start+i].call
	}
	plan, err := planner.ParallelBatch(ctx, s.runner.Settings, batch[:count], maxParallelTools)
	if err != nil {
		return plan, err
	}
	if plan.Count < 1 || plan.Count > count || plan.PreparationConcurrency < 1 || plan.PreparationConcurrency > min(4, plan.Count) {
		return plan, faults.Errorf(diagcatalog.RunInvalidParallelBatchBoundary)
	}
	return plan, nil
}

func fatalBatchError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, errApprovalDenied) || errors.Is(err, errRuntimeRecord) || faults.IsPanic(err) {
		return err
	}
	return nil
}
