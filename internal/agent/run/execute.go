// Purpose: Execute one approved operation while sharing preparation and result recording with batches.
// Pattern: Tool execution use case.
// Usage: action invokes execute; batch uses the same prepare/start/record boundaries.
// Related: internal/agent/run/prepare_execution.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"errors"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

var errRuntimeRecord = errors.New("runtime observation could not be committed or delivered")

func (s *execution) execute(ctx context.Context, call toolapi.Call, approve Approver, emit func(Event) error) (toolapi.Result, error) {
	ready, outcome, err := s.prepareExecution(ctx, call, approve, emit)
	if err != nil {
		return outcome, errors.Join(err, s.recordRejected(ctx, call, outcome, err, emit))
	}
	ctx, cancel := context.WithDeadline(ctx, s.usage.snapshot().deadline)
	defer cancel()
	if err := s.startExecution(ctx, ready, emit); err != nil {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	outcome, err = ready.operation.Execute(ctx, ready.approved)
	return s.recordExecution(ctx, ready, outcome, err, emit)
}
func (s *execution) startExecution(ctx context.Context, ready *readyExecution, emit func(Event) error) error {
	s.checkpoint.Phase = "execution"
	if err := s.persist(ctx); err != nil {
		return err
	}
	event := Event{Kind: "tool_started", Text: ready.proposal.Name + " " + ready.proposal.Target, CallID: ready.call.ID, Call: &ready.call}
	if err := s.record(ctx, event); err != nil {
		return err
	}
	return emit(event)
}
