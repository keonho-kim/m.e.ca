// Purpose: Prepare, review and approve a grounded operation on the owning runtime worker.
// Pattern: Approval use case.
// Usage: Single and batch execution call prepareExecution before any concurrent execution.
// Related: internal/agent/run/conclave.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

type readyExecution struct {
	grounding grounding.Record
	call      toolapi.Call
	operation toolapi.Execution
	proposal  toolapi.Approval
	review    *conclave.Result
	approved  bool
}

func (s *execution) freezeExecution(ctx context.Context, call toolapi.Call, emit func(Event) error) (ready *readyExecution, outcome toolapi.Result, err error) {
	if s.checkpoint.Grounding == nil || !s.checkpoint.Grounding.Matches(call) {
		return nil, outcome, fmt.Errorf("missing self-grounding for exact tool call")
	}
	if err := s.checkTimeBudget(ctx); err != nil {
		return nil, outcome, err
	}
	if call.Name == "write_file" || call.Name == "edit_file" {
		s.checkpoint.PendingReview = true
	}
	r := s.runner
	op, err := r.Tools.Prepare(ctx, r.Settings, call)
	if err != nil {
		outcome.Failure = toolapi.FailureOf(err)
		event := Event{Kind: "tool_failed", Text: call.Name + ": " + r.Redact(outcome.Failure.Error()), CallID: call.ID, Call: &call}
		if recordErr := s.record(context.WithoutCancel(ctx), event); recordErr != nil {
			return nil, outcome, errors.Join(errRuntimeRecord, recordErr)
		}
		if emitErr := emit(event); emitErr != nil {
			return nil, outcome, errors.Join(errRuntimeRecord, emitErr)
		}
		return nil, outcome, err
	}
	proposal := op.Approval()
	outcome.Target = proposal.Target
	return &readyExecution{call: call, operation: op, proposal: proposal, grounding: *s.checkpoint.Grounding}, outcome, nil
}

func (s *execution) prepareExecution(ctx context.Context, call toolapi.Call, approve Approver, emit func(Event) error) (*readyExecution, toolapi.Result, error) {
	ready, outcome, err := s.freezeExecution(ctx, call, emit)
	if err != nil {
		return nil, outcome, err
	}
	return s.approveExecution(ctx, ready, nil, approve, emit)
}

func (s *execution) approveExecution(ctx context.Context, ready *readyExecution, pending []toolapi.FileDelta, approve Approver, emit func(Event) error) (*readyExecution, toolapi.Result, error) {
	call, proposal := ready.call, ready.proposal
	outcome := toolapi.Result{Target: proposal.Target}
	s.checkpoint.Grounding = &ready.grounding
	var err error
	review, reviewErr := s.reviewCandidate(ctx, call, proposal, pending)
	if reviewErr != nil {
		outcome.Failed = true
		outcome.Failure = toolapi.FailureOf(reviewErr)
		if outcome.Failure.Code != reviewRejected && outcome.Failure.Code != reviewFailed {
			return nil, outcome, errors.Join(errRuntimeRecord, reviewErr)
		}
		return nil, outcome, reviewErr
	}
	if proposal.Candidate != nil && proposal.Candidate.Lines() == 0 {
		s.checkpoint.PendingReview = false
	}
	proposal.ConclavePassed = review != nil && review.Passed
	if err := s.checkTimeBudget(ctx); err != nil {
		return nil, outcome, err
	}
	ctx, cancel := context.WithDeadline(ctx, s.usage.deadline)
	defer cancel()
	approved := !proposal.Required
	if proposal.Required {
		approved, err = approve(ctx, proposal)
		if err != nil {
			return nil, outcome, errors.Join(errRuntimeRecord, err)
		}
		decision := "denied"
		if approved {
			decision = "approved"
		}
		event := Event{Kind: decision, Text: decision + " " + proposal.Name + " " + proposal.Target, CallID: call.ID, Approval: &proposal, Call: &call, Conclave: review}
		if err := s.record(ctx, event); err != nil {
			return nil, outcome, errors.Join(errRuntimeRecord, err)
		}
		if err := emit(event); err != nil {
			return nil, outcome, errors.Join(errRuntimeRecord, err)
		}
		if !approved {
			return nil, outcome, errApprovalDenied
		}
	}

	ready.proposal, ready.review, ready.approved = proposal, review, approved
	return ready, outcome, nil
}
