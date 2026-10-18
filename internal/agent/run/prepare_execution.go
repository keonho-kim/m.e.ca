// Purpose: Prepare and approve a grounded operation on the owning runtime worker.
// Pattern: Approval use case.
// Usage: Single and batch execution call prepareExecution before any concurrent execution.
// Related: internal/agent/run/grounding.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

type readyExecution struct {
	beforeVersions map[string]string
	grounding      grounding.Record
	call           toolapi.Call
	operation      toolapi.Execution
	proposal       toolapi.Approval
	approved       bool
}

func (s *execution) freezeExecution(ctx context.Context, call toolapi.Call) (ready *readyExecution, outcome toolapi.Result, err error) {
	if s.checkpoint.Grounding == nil || !s.checkpoint.Grounding.Matches(call) {
		return nil, outcome, fmt.Errorf("missing self-grounding for exact tool call")
	}
	if err := s.checkTimeBudget(ctx); err != nil {
		return nil, outcome, err
	}
	r := s.runner
	op, err := r.Tools.Prepare(ctx, r.Settings, call)
	if err != nil {
		outcome.Failure = toolapi.FailureOf(err)

		return nil, outcome, err
	}
	proposal := op.Approval()
	if proposal.Candidate != nil {
		if err := proposal.Candidate.Check(call); err != nil {
			return nil, outcome, err
		}
	}
	outcome.Target = proposal.Target
	return &readyExecution{call: call, operation: op, proposal: proposal, grounding: *s.checkpoint.Grounding}, outcome, nil
}

func (s *execution) prepareExecution(ctx context.Context, call toolapi.Call, approve Approver, emit func(Event) error) (*readyExecution, toolapi.Result, error) {
	ready, outcome, err := s.freezeExecution(ctx, call)
	if err != nil {
		return nil, outcome, err
	}
	return s.approveExecution(ctx, ready, approve, emit)
}

func (s *execution) approveExecution(ctx context.Context, ready *readyExecution, approve Approver, emit func(Event) error) (*readyExecution, toolapi.Result, error) {
	call, proposal := ready.call, ready.proposal
	outcome := toolapi.Result{Target: proposal.Target}
	s.checkpoint.Grounding = &ready.grounding
	var err error
	if err := s.checkTimeBudget(ctx); err != nil {
		return nil, outcome, err
	}
	ctx, cancel := context.WithDeadline(ctx, s.usage.snapshot().deadline)
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
		event := Event{Kind: decision, Text: decision + " " + proposal.Name + " " + proposal.Target, CallID: call.ID, Approval: &proposal, Call: &call}
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

	ready.proposal, ready.approved = proposal, approved
	return ready, outcome, nil
}
