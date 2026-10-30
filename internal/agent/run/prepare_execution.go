// Purpose: Prepare and approve an operation on the owning runtime worker.
// Pattern: Approval use case.
// Usage: Single and batch execution call prepareExecution before any concurrent execution.
// Related: internal/agent/run/call_validation.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

type readyExecution struct {
	call      toolapi.Call
	operation toolapi.Execution
	proposal  toolapi.Approval
	approved  bool
}

func (s *execution) freezeExecution(ctx context.Context, call toolapi.Call) (ready *readyExecution, outcome toolapi.Result, err error) {
	if err := s.checkTimeBudget(ctx); err != nil {
		return nil, outcome, err
	}
	r := s.runner
	if call.Name == "read_image" && !r.ImageInput {
		return nil, outcome, &modelapi.ImageInputError{}
	}
	op, err := r.Tools.Prepare(ctx, r.Settings, call)
	if err != nil {
		outcome.Failure = toolapi.FailureOf(err)

		return nil, outcome, err
	}
	proposal := op.Approval()
	proposal.Required = proposal.Required || proposal.UserRequired
	if proposal.Candidate != nil {
		if err := proposal.Candidate.Check(call); err != nil {
			return nil, outcome, err
		}
	}
	outcome.Target = proposal.Target
	return &readyExecution{call: call, operation: op, proposal: proposal}, outcome, nil
}

func (s *execution) prepareExecution(ctx context.Context, call toolapi.Call, approve Approver, emit func(runrecord.Event) error) (*readyExecution, toolapi.Result, error) {
	ready, outcome, err := s.freezeExecution(ctx, call)
	if err != nil {
		return nil, outcome, err
	}
	return s.approveExecution(ctx, ready, approve, emit)
}

func (s *execution) approveExecution(ctx context.Context, ready *readyExecution, approve Approver, emit func(runrecord.Event) error) (*readyExecution, toolapi.Result, error) {
	call, proposal := ready.call, ready.proposal
	outcome := toolapi.Result{Target: proposal.Target}
	var err error
	if err := s.checkTimeBudget(ctx); err != nil {
		return nil, outcome, err
	}
	ctx, cancel := context.WithDeadline(ctx, s.usage.snapshot().deadline)
	defer cancel()
	approved := !proposal.Required
	if proposal.Required {
		source := "user"
		if s.runner.Settings.Permission == policy.PermissionAllMighty && !proposal.UserRequired {
			approved, source = true, "all_mighty"
		} else {
			approved, err = approve(ctx, proposal)
			if err != nil {
				return nil, outcome, errors.Join(errApprovalDenied, err)
			}
		}
		decision := "denied"
		if approved {
			decision = "approved"
		}
		event := runrecord.Event{Kind: decision, Text: decision + " " + proposal.Name + " " + proposal.Target, CallID: call.ID, Approval: &proposal, Call: &call, ApprovalSource: source}
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
