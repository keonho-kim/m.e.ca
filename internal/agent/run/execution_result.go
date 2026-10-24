// Purpose: Commit actual outcomes, native replies and checkpoints before publishing them.
// Pattern: Unit of Work on the serial runtime owner.
// Usage: Workers return outcomes; single and batch owners call recordExecution serially.
// Related: internal/agent/run/execute.go, internal/agent/run/batch.go.
package run

import (
	"context"
	"errors"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"time"
)

func (s *execution) recordExecution(ctx context.Context, v validatedCall, ready *readyExecution, outcome toolapi.Result, runErr error, emit func(Event) error) (toolapi.Result, error) {
	r, call, proposal := s.runner, ready.call, ready.proposal
	event := Event{CallID: call.ID, Call: &call}
	if call.Name != "execute_command" || outcome.Process == nil {
		outcome.Started = true
	}
	outcome.Target = proposal.Target
	event.Kind, event.Text = "tool_succeeded", outcome.Summary
	if outcome.Failed {
		event.Kind = "tool_failed"
	}
	if runErr != nil {
		if outcome.Failure == nil {
			outcome.Failure = toolapi.FailureOf(runErr)
		}
		outcome.Failed = true
		event.Kind = "tool_failed"
	}
	if outcome.Failure != nil {
		// The local stale-version guard proves it rejected the operation before mutation.
		if outcome.Failure.Code == toolapi.StaleTarget {
			outcome.Started = false
		}
		outcome.Failure.Detail = r.Redact(outcome.Failure.Detail)
		outcome.Failure.Next = r.Redact(outcome.Failure.Next)
		event.Text = outcome.Failure.Error()
	}
	opFact := toolapi.Operation{Target: outcome.Target, Versions: outcome.Versions, Started: outcome.Started, RunID: s.checkpoint.ID, CallID: call.ID, Name: call.Name, Status: event.Kind, Changes: outcome.Changes, Result: event.Text, Failure: outcome.Failure, Process: outcome.Process}
	opFact.Revision = s.checkpoint.ChangeRevision
	if call.Name == "execute_command" {
		opFact.Command = proposal.Target
	}
	event.Operation = &opFact
	event.Output = outcome.Content
	event.Capture = outcome.Capture
	if event.Kind == "tool_succeeded" && proposal.Diff {
		event.ChangePreview = boundedChangePreview(r.Redact(proposal.Preview))
	}
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.acceptResult(recordCtx, v, outcome, runErr, event); err != nil {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	event.Capture = nil // Original captures are durable, never copied into live presentation events.
	if err := emit(event); err != nil && !(ctx.Err() != nil && errors.Is(err, ctx.Err())) {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	return outcome, runErr
}
