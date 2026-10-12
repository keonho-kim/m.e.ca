// Purpose: Execute one complete tool through existing policy, approval and durable event boundaries.
// Pattern: Use Case.
// Usage: Called only after every staged argument is parsed and the call is frozen.
// Related: internal/agent/run/action.go, internal/agent/tools.go.
package run

import (
	"context"

	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/compact"
)

func (s *task) execute(ctx context.Context, planning bool, call agent.ToolCall, approve Approver, emit func(Event) error) (string, bool, error) {
	r := s.runner
	op, err := r.Tools.Prepare(ctx, r.Settings, planning, call)
	if err != nil {
		event := Event{Kind: "tool_failed", Text: call.Name + ": " + r.Redact(err.Error()), CallID: call.ID}
		if recordErr := r.Journal.RecordEvent(context.WithoutCancel(ctx), r.Session, event); recordErr != nil {
			return "", false, recordErr
		}
		if emitErr := emit(event); emitErr != nil {
			return "", false, emitErr
		}
		return "", false, err
	}
	proposal := op.Approval()
	approved := !proposal.Required
	if proposal.Required {
		approved, err = approve(ctx, proposal)
		if err != nil {
			return "", false, err
		}
		decision := "denied"
		if approved {
			decision = "approved"
		}
		event := Event{Kind: decision, Text: decision + " " + proposal.Name + " " + proposal.Target, CallID: call.ID, Approval: &proposal}
		if err = r.Journal.RecordEvent(ctx, r.Session, event); err != nil {
			return "", false, err
		}
		if err = emit(event); err != nil {
			return "", false, err
		}
		if !approved {
			return "", false, errApprovalDenied
		}
	}
	s.checkpoint.Phase = "execution"
	if err = s.persist(ctx); err != nil {
		return "", false, err
	}
	event := Event{Kind: "tool_started", Text: proposal.Name + " " + proposal.Target, CallID: call.ID}
	if err = r.Journal.RecordEvent(ctx, r.Session, event); err != nil {
		return "", false, err
	}
	if err = emit(event); err != nil {
		return "", false, err
	}
	outcome, runErr := op.Execute(ctx, approved)
	event.Kind, event.Text = "tool_succeeded", outcome.Summary
	if outcome.Failed {
		event.Kind = "tool_failed"
	}
	if runErr != nil {
		event.Kind, event.Text = "tool_failed", r.Redact(runErr.Error())
	}
	opFact := compact.Operation{TaskID: s.checkpoint.ID, CallID: call.ID, Name: call.Name, Status: event.Kind, Check: s.checkpoint.Checking, Step: s.runner.Step, Changes: outcome.Changes, Result: event.Text}
	if call.Name == "execute_command" {
		opFact.Command = proposal.Target
	}
	event.Operation = &opFact
	// Cancellation must still leave an interrupted event on disk.
	recordCtx := context.WithoutCancel(ctx)
	if err = r.Journal.RecordEvent(recordCtx, r.Session, event); err != nil {
		return "", false, err
	}
	if err = emit(event); err != nil {
		return "", false, err
	}
	return string(outcome.Content), outcome.Failed, runErr
}
