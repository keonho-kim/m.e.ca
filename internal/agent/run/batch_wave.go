// Purpose: Execute independent operations concurrently and commit their ready prefix in call order.
// Pattern: Fork/join with isolated result slots.
// Usage: batch schedules at most sixteen operations; workers touch only their prepared operation.
// Related: internal/agent/run/batch.go, internal/agent/run/prepare_execution.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"sync"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) batchWave(ctx context.Context, calls []validatedCall, approve Approver) (bool, error) {
	ctx, cancel := context.WithDeadline(ctx, s.usage.snapshot().deadline)
	defer cancel()
	ready := make([]*readyExecution, len(calls))
	for i, v := range calls {
		s.checkpoint.Tool, s.checkpoint.Arguments = v.call.Name, v.arguments
		if err := s.record(ctx, runrecord.Event{Kind: "tool_proposed", CallID: v.call.ID, Call: &v.call, Text: "Native candidate; not executed"}); err != nil {
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
	completed := make(chan int, len(calls))
	var workers sync.WaitGroup
	for i, prepared := range ready {
		workers.Go(func() {
			outcomes[i], failures[i] = prepared.operation.Execute(ctx, prepared.approved)
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
			if errors.Is(runErr, errRuntimeRecord) {
				terminal = errors.Join(terminal, runErr)
			}
			failed = failed || outcome.Failed || runErr != nil
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
