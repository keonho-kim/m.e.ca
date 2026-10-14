// Purpose: Execute a grounded PFC call through policy, approval and durable observation boundaries.
// Pattern: Tool execution use case.
// Usage: action invokes execute after fields and self-grounding have been persisted.
// Related: internal/agent/run/action.go, internal/agent/tool/execution.go, internal/agent/verification/state.go.
package run

import (
	"context"
	"errors"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

var errRuntimeRecord = errors.New("runtime observation could not be committed or delivered")

func (s *task) execute(ctx context.Context, planning bool, call toolapi.Call, approve Approver, emit func(Event) error) (outcome toolapi.Result, err error) {
	if s.checkpoint.Grounding == nil || !s.checkpoint.Grounding.Matches(call) {
		return outcome, fmt.Errorf("missing self-grounding for exact tool call")
	}
	if err := s.checkTimeBudget(ctx); err != nil {
		return outcome, err
	}
	if call.Name == "write_file" || call.Name == "edit_file" {
		s.checkpoint.PendingReview = true
	}
	r := s.runner
	op, err := r.Tools.Prepare(ctx, r.Settings, planning, call)
	if err != nil {
		outcome.Failure = toolapi.FailureOf(err)
		event := Event{Kind: "tool_failed", Text: call.Name + ": " + r.Redact(outcome.Failure.Error()), CallID: call.ID, Call: &call}
		if recordErr := s.record(context.WithoutCancel(ctx), event); recordErr != nil {
			return outcome, errors.Join(errRuntimeRecord, recordErr)
		}
		if emitErr := emit(event); emitErr != nil {
			return outcome, errors.Join(errRuntimeRecord, emitErr)
		}
		return outcome, err
	}
	proposal := op.Approval()
	outcome.Target = proposal.Target
	review, reviewErr := s.reviewCandidate(ctx, call, proposal)
	if reviewErr != nil {
		outcome.Failed = true
		outcome.Failure = toolapi.FailureOf(reviewErr)
		if outcome.Failure.Code != reviewRejected && outcome.Failure.Code != reviewFailed {
			return outcome, errors.Join(errRuntimeRecord, reviewErr)
		}
		return outcome, reviewErr
	}
	proposal.ConclavePassed = review != nil && review.Passed
	if err := s.checkTimeBudget(ctx); err != nil {
		return outcome, err
	}
	ctx, cancel := context.WithDeadline(ctx, s.usage.deadline)
	defer cancel()
	approved := !proposal.Required
	if proposal.Required {
		approved, err = approve(ctx, proposal)
		if err != nil {
			return outcome, errors.Join(errRuntimeRecord, err)
		}
		decision := "denied"
		if approved {
			decision = "approved"
		}
		event := Event{Kind: decision, Text: decision + " " + proposal.Name + " " + proposal.Target, CallID: call.ID, Approval: &proposal, Call: &call, Conclave: review}
		if err := s.record(ctx, event); err != nil {
			return outcome, errors.Join(errRuntimeRecord, err)
		}
		if err := emit(event); err != nil {
			return outcome, errors.Join(errRuntimeRecord, err)
		}
		if !approved {
			return outcome, errApprovalDenied
		}
	}
	s.checkpoint.Phase = "execution"
	if err := s.persist(ctx); err != nil {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	event := Event{Kind: "tool_started", Text: proposal.Name + " " + proposal.Target, CallID: call.ID, Call: &call, Conclave: review}
	if err := s.record(ctx, event); err != nil {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	if err := emit(event); err != nil {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	outcome, runErr := op.Execute(ctx, approved)
	outcome.Started = true
	outcome.ReviewSatisfied = proposal.Candidate != nil && (proposal.Candidate.Lines() == 0 || review != nil && review.Passed)
	outcome.Target = proposal.Target
	event.Kind, event.Text = "tool_succeeded", outcome.Summary
	if outcome.Failed {
		event.Kind = "tool_failed"
	}
	if runErr != nil {
		outcome.Failure = toolapi.FailureOf(runErr)
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
	opFact := toolapi.Operation{TaskID: s.checkpoint.ID, CallID: call.ID, Name: call.Name, Status: event.Kind, Check: s.checkpoint.Checking, Step: s.runner.Step, Changes: outcome.Changes, Result: event.Text, Failure: outcome.Failure, Process: outcome.Process}
	if call.Name == "execute_command" {
		opFact.Command = proposal.Target
	}
	event.Operation = &opFact
	event.Output = outcome.Content
	recordCtx := context.WithoutCancel(ctx)
	if err := s.record(recordCtx, event); err != nil {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	if err := emit(event); err != nil {
		return outcome, errors.Join(errRuntimeRecord, err)
	}
	return outcome, runErr
}
