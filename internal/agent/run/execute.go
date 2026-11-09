// Purpose: Execute one approved operation while sharing preparation and result recording with batches.
// Pattern: Tool execution use case.
// Usage: action invokes execute; batch uses the same prepare/start/record boundaries.
// Related: internal/agent/run/prepare_execution.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"errors"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

var errRuntimeRecord = faults.Errorf(diagcatalog.RunRuntimeObservationCouldNotBeCommittedOr)

func (s *execution) execute(ctx context.Context, v validatedCall, approve Approver, emit func(runrecord.Event) error) (toolapi.Result, error) {
	ctx = instruction.WithExplicitReads(ctx, s.explicitSkills)
	instructionRevision := s.instructionRevision
	ready, outcome, err := s.prepareExecution(ctx, v.call, approve, emit)
	if err != nil {
		return outcome, errors.Join(err, s.recordRejected(ctx, v, outcome, err, emit))
	}
	defer toolapi.Release(ready.operation)
	if err := s.checkSteering(ctx, instructionRevision); err != nil {
		outcome.Failure = faults.FailureOf(err)
		return outcome, errors.Join(err, s.recordRejected(ctx, v, outcome, err, emit))
	}
	ctx, cancel := context.WithDeadline(ctx, s.usage.deadline)
	defer cancel()
	if err := s.startExecution(ctx, ready, emit); err != nil {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	outcome, err = executePrepared(ctx, ready)
	return s.recordExecution(ctx, v, ready, outcome, err, emit)
}

func executePrepared(ctx context.Context, ready *readyExecution) (outcome toolapi.Result, err error) {
	err = faults.Guard(func() error {
		var cause error
		outcome, cause = ready.operation.Execute(ctx, ready.approved)
		return cause
	})
	if faults.IsPanic(err) {
		outcome.Started = true
		outcome.Failed = true
		outcome.Failure = faults.FailureOf(err)
	}
	return outcome, err
}
func (s *execution) startExecution(ctx context.Context, ready *readyExecution, emit func(runrecord.Event) error) error {
	return s.startExecutions(ctx, []*readyExecution{ready}, emit)
}

func (s *execution) startExecutions(ctx context.Context, ready []*readyExecution, emit func(runrecord.Event) error) error {
	s.checkpoint.Phase = "execution"
	events := make([]runrecord.Event, len(ready))
	for i, value := range ready {
		events[i] = runrecord.Event{Kind: "tool_started", Text: value.proposal.Name + " " + value.proposal.Target, CallID: value.call.ID, Call: &value.call, RunID: s.checkpoint.ID, OccurredAt: time.Now().UTC().Format(time.RFC3339Nano)}
	}
	if err := s.persist(ctx, events...); err != nil {
		return err
	}
	for _, event := range events {
		if err := emit(event); err != nil {
			return err
		}
	}
	return nil
}
