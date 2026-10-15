// Purpose: Prepare a wave on its owner, execute independent operations concurrently, then commit in order.
// Pattern: Fork/join with isolated result slots.
// Usage: batch schedules at most four operations; workers touch only their prepared operation.
// Related: internal/agent/run/batch.go, internal/agent/run/prepare_execution.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"errors"
	"sync"
	"time"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s *execution) batchWave(ctx context.Context, calls []validatedCall, approve Approver) (bool, error) {
	ctx, cancel := context.WithDeadline(ctx, s.usage.deadline)
	defer cancel()
	ready := make([]*readyExecution, len(calls))
	for i, v := range calls {
		if err := s.activateCall(ctx, v); err != nil {
			return true, errors.Join(s.skipCalls(ctx, calls, err.Error()), fatalBatchError(ctx, err))
		}
		prepared, _, err := s.freezeExecution(ctx, v.call, s.emit)
		if err != nil {
			return true, errors.Join(s.skipCalls(ctx, calls, err.Error()), fatalBatchError(ctx, err))
		}
		ready[i] = prepared
	}
	// Every candidate is frozen before any review. Review dependencies can inspect
	// sibling proposals without pretending their files have already been written.
	for i, prepared := range ready {
		var pending []toolapi.FileDelta
		for j, sibling := range ready {
			if i != j && sibling.proposal.Candidate != nil {
				pending = append(pending, sibling.proposal.Candidate.Files...)
			}
		}
		approved, _, err := s.approveExecution(ctx, prepared, pending, approve, s.emit)
		if err != nil {
			return true, errors.Join(s.skipCalls(ctx, calls, err.Error()), fatalBatchError(ctx, err))
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
	pendingReview := s.checkpoint.PendingReview
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
	// A successful sibling cannot erase an unresolved review from another operation.
	s.checkpoint.PendingReview = s.checkpoint.PendingReview || pendingReview
	if ctx.Err() != nil {
		terminal = errors.Join(terminal, ctx.Err())
		failed = true
	}
	return failed, terminal
}
