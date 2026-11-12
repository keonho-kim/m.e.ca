// Purpose: Execute independent operations concurrently and commit their ready prefix in call order.
// Pattern: Fork/join with isolated result slots.
// Usage: batch schedules at most sixteen operations; workers touch only their prepared operation.
// Related: internal/agent/run/batch.go, internal/agent/run/prepare_execution.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"errors"
	"sync"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func (s *execution) batchWave(ctx context.Context, calls []validatedCall, preparation int, approve Approver) (bool, error) {
	ctx = instruction.WithExplicitReads(ctx, s.explicitSkills)
	ctx, cancel := context.WithDeadline(ctx, s.usage.deadline)
	defer cancel()
	instructionRevision := s.instructionRevision
	if len(calls) > 1 {
		batchID := "batch_" + calls[0].call.ExecutionID
		for i := range calls {
			calls[i].call.BatchID = batchID
		}
	}
	ready, index, outcome, err := s.freezeWave(ctx, calls, preparation)
	if err != nil {
		return true, errors.Join(s.rejectWave(ctx, calls, index, outcome, err), fatalBatchError(ctx, err))
	}
	defer releaseReady(ready)
	// Every candidate is frozen before approval; all workers start only after
	// the owner has approved and rechecked the complete wave.
	for i, prepared := range ready {
		approved, outcome, err := s.approveExecution(ctx, prepared, approve, s.emit)
		if err != nil {
			return true, errors.Join(s.rejectWave(ctx, calls, i, outcome, err), fatalBatchError(ctx, err))
		}
		ready[i] = approved
	}
	if err := s.checkSteering(ctx, instructionRevision); err != nil {
		var failure *faults.Failure
		if errors.As(err, &failure) && failure.Code == diagcatalog.InstructionChanged {
			return true, s.skipCalls(ctx, calls, skipForInstruction, failure.Detail)
		}
		return true, errors.Join(err, s.skipCalls(ctx, calls, skipForContext, err.Error()))
	}
	// Approval may have changed filesystem aliases; recheck the wave before effects.
	commandsOnly := true
	for _, call := range calls {
		commandsOnly = commandsOnly && call.call.Name == "execute_command"
	}
	if !commandsOnly {
		if plan, err := s.planWave(ctx, calls, 0); err != nil || plan.Count != len(calls) {
			detail := "Batch targets changed or overlap after preparation; nothing in this wave executed."
			if err != nil {
				detail += " " + err.Error()
			}
			return true, errors.Join(s.skipCalls(ctx, calls, skipAfterFailure, detail), fatalBatchError(ctx, err))
		}
	}
	// Persist start intent before allowing any effects. A crash never replays it automatically.
	if err := s.checkTimeBudget(ctx); err != nil {
		return true, errors.Join(s.skipCalls(ctx, calls, skipForInterruption, err.Error()), err)
	}
	if err := s.startExecutions(ctx, ready, s.emit); err != nil {
		return true, errors.Join(s.skipCalls(ctx, calls, skipForContext, err.Error()), errRuntimeRecord, err)
	}
	outcomes := make([]toolapi.Result, len(calls))
	failures := make([]error, len(calls))
	completed := make(chan int, len(calls))
	var workers sync.WaitGroup
	for i, prepared := range ready {
		workers.Go(func() {
			outcomes[i], failures[i] = executePrepared(ctx, prepared)
			completed <- i
		})
	}
	completedEmit := func(event runrecord.Event) error {
		err := s.emit(event)
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return nil
		}
		return err
	}
	failed := false
	var terminal error
	finished := make([]bool, len(calls))
	next := 0
	for range calls {
		finished[<-completed] = true
		// Preserve native reply order without waiting for later unrelated calls.
		for next < len(calls) && finished[next] {
			outcome, runErr := s.recordExecution(ctx, calls[next], ready[next], outcomes[next], failures[next], completedEmit)
			if errors.Is(runErr, errRuntimeRecord) || faults.IsPanic(runErr) {
				terminal = errors.Join(terminal, runErr)
			}
			failed = failed || outcome.Failed || runErr != nil
			toolapi.Release(ready[next].operation)
			ready[next], outcomes[next], failures[next] = nil, toolapi.Result{}, nil
			next++
		}
	}
	// Join every worker, including after cancellation or a failed result commit.
	workers.Wait()
	if ctx.Err() != nil {
		terminal = errors.Join(terminal, ctx.Err())
		failed = true
	}
	return failed, terminal
}
