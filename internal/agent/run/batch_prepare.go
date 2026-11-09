// Purpose: Freeze independent file candidates with bounded workers and deterministic failure selection.
// Pattern: Fork/join preparation with one journal owner.
// Usage: batchWave calls this before any approvals, start intents or filesystem changes.
// Related: internal/tools/local/parallel.go, internal/agent/run/prepare_execution.go.
package run

import (
	"context"
	"sync"
	"sync/atomic"

	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) freezeWave(ctx context.Context, calls []validatedCall, count int) ([]*readyExecution, int, toolapi.Result, error) {
	ready := make([]*readyExecution, len(calls))
	complete := false
	defer func() {
		if !complete {
			releaseReady(ready)
		}
	}()
	proposals := make([]runrecord.Event, len(calls))
	for i, v := range calls {
		proposals[i] = runrecord.Event{Kind: "tool_proposed", CallID: v.call.ID, Call: &calls[i].call, Text: "Native candidate; not executed"}
	}
	if err := s.recordEvents(ctx, proposals); err != nil {
		return nil, 0, toolapi.Result{}, err
	}
	for i, v := range calls {
		s.checkpoint.Tool = v.call.Name
		if err := s.checkTimeBudget(ctx); err != nil {
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
		complete = true
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
				failures[i] = faults.Guard(func() error { var err error; ready[i], _, err = s.freezeExecution(ctx, calls[i].call); return err })
			}
		})
	}
	workers.Wait()
	for i, err := range failures {
		if err != nil {
			return nil, i, toolapi.Result{}, err
		}
	}
	complete = true
	return ready, -1, toolapi.Result{}, nil
}

func releaseReady(ready []*readyExecution) {
	for _, value := range ready {
		if value != nil {
			toolapi.Release(value.operation)
		}
	}
}
