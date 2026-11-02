// Purpose: Freeze independent file candidates with bounded workers and deterministic failure selection.
// Pattern: Fork/join preparation with one journal owner.
// Usage: batchWave calls this before any approvals, start intents or filesystem changes.
// Related: internal/tools/local/parallel.go, internal/agent/run/prepare_execution.go.
package run

import (
	"context"
	"sync"
	"sync/atomic"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) freezeWave(ctx context.Context, calls []validatedCall, count int) ([]*readyExecution, int, toolapi.Result, error) {
	ready := make([]*readyExecution, len(calls))
	for i, v := range calls {
		s.checkpoint.Tool = v.call.Name
		if err := s.checkTimeBudget(ctx); err != nil {
			return nil, i, toolapi.Result{}, err
		}
		if err := s.record(ctx, runrecord.Event{Kind: "tool_proposed", CallID: v.call.ID, Call: &v.call, Text: "Native candidate; not executed"}); err != nil {
			return nil, i, toolapi.Result{}, err
		}
		if count == 1 {
			prepared, outcome, err := s.freezeExecution(ctx, v.call)
			if err != nil {
				return nil, i, outcome, err
			}
			ready[i] = prepared
		}
	}
	if count == 1 {
		return ready, -1, toolapi.Result{}, nil
	}
	failures := make([]error, len(calls))
	var next atomic.Int64
	var workers sync.WaitGroup
	for range count {
		workers.Go(func() {
			for {
				i := int(next.Add(1)) - 1
				if i >= len(calls) {
					return
				}
				ready[i], _, failures[i] = s.freezeExecution(ctx, calls[i].call)
			}
		})
	}
	workers.Wait()
	for i, err := range failures {
		if err != nil {
			return nil, i, toolapi.Result{}, err
		}
	}
	return ready, -1, toolapi.Result{}, nil
}
