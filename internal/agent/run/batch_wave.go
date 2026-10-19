// Purpose: Prepare a wave on its owner, execute independent operations concurrently, then commit in order.
// Pattern: Fork/join with isolated result slots.
// Usage: batch schedules at most four operations; workers touch only their prepared operation.
// Related: internal/agent/run/batch.go, internal/agent/run/prepare_execution.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s *execution) batchWave(ctx context.Context, calls []validatedCall, approve Approver) (bool, error) {
	ctx, cancel := context.WithDeadline(ctx, s.usage.snapshot().deadline)
	defer cancel()
	ready := make([]*readyExecution, len(calls))
	for i, v := range calls {
		s.checkpoint.Tool, s.checkpoint.Arguments = v.call.Name, v.arguments
		if err := s.record(ctx, Event{Kind: "tool_proposed", CallID: v.call.ID, Call: &v.call, Text: "Native candidate; not executed"}); err != nil {
			return true, errors.Join(s.skipCalls(ctx, calls, fmt.Sprintf("Call %s (%s) was not prepared: %s", v.call.ID, v.call.Name, err)), fatalBatchError(ctx, err))
		}
		prepared, outcome, err := s.freezeExecution(ctx, v.call)
		if err != nil {
			return true, errors.Join(s.rejectWave(ctx, calls, i, outcome, err), fatalBatchError(ctx, err))
		}
		ready[i] = prepared
	}
	// Every candidate is frozen before approval; all workers start only after
	// the owner has approved and rechecked the complete wave.
	for i, prepared := range ready {
		approved, outcome, err := s.approveExecution(ctx, prepared, approve, s.emit)
		if err != nil {
			return true, errors.Join(s.rejectWave(ctx, calls, i, outcome, err), fatalBatchError(ctx, err))
		}
		ready[i] = approved
	}
	// Approval may have changed filesystem aliases; recheck the wave before effects.
	if end, err := s.waveEnd(ctx, calls, 0); err != nil || end != len(calls) {
		detail := "Batch targets changed or overlap after preparation; nothing in this wave executed."
		if err != nil {
			detail += " " + err.Error()
		}
		return true, errors.Join(s.skipCalls(ctx, calls, detail), fatalBatchError(ctx, err))
	}
	// Persist start intent before allowing any effects. A crash never replays it automatically.
	for _, prepared := range ready {
		if err := s.checkTimeBudget(ctx); err != nil {
			return true, errors.Join(s.skipCalls(ctx, calls, err.Error()), err)
		}
		if err := s.startExecution(ctx, prepared, s.emit); err != nil {
			return true, errors.Join(s.skipCalls(ctx, calls, err.Error()), errRuntimeRecord, err)
		}
	}
	outcomes := make([]toolapi.Result, len(calls))
	failures := make([]error, len(calls))
	var workers sync.WaitGroup
	for i, prepared := range ready {
		workers.Go(func() { outcomes[i], failures[i] = prepared.operation.Execute(ctx, prepared.approved) })
	}
	workers.Wait()
	// Workers are joined even after cancellation; completed effects keep their actual results.
	recordCtx, recordCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer recordCancel()
	completedEmit := func(event Event) error {
		err := s.emit(event)
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return nil
		}
		return err
	}
	failed := false
	var terminal error
	for i, v := range calls {
		outcome, runErr := s.recordExecution(recordCtx, ready[i], outcomes[i], failures[i], completedEmit)
		if errors.Is(runErr, errRuntimeRecord) {
			terminal = errors.Join(terminal, runErr)
		}
		failed = failed || outcome.Failed || runErr != nil
		if err := s.acceptResult(recordCtx, v, outcome, runErr); err != nil {
			terminal = errors.Join(terminal, err)
		}
	}
	if ctx.Err() != nil {
		terminal = errors.Join(terminal, ctx.Err())
		failed = true
	}
	return failed, terminal
}
