// Purpose: Execute prepared operations and persist start intent before worker launch.
// Pattern: Tool execution use case.
// Usage: Code scheduling starts approved groups; workers return outcomes to the serial owner.
// Related: internal/agent/run/prepare_execution.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var errRuntimeRecord = faults.Errorf(diagcatalog.RunRuntimeObservationCouldNotBeCommittedOr)

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
