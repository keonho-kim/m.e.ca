// Purpose: Prepare and approve an operation on the owning runtime worker.
// Pattern: Approval use case.
// Usage: Single and batch execution call prepareExecution before any concurrent execution.
// Related: internal/agent/run/call_validation.go, internal/agent/run/execute.go.
package run

import (
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"context"
	"errors"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
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
	if call.Name == "view_image" && !r.ImageInput {
		return nil, outcome, &modelapi.ImageInputError{}
	}
	var op toolapi.Execution
	err = faults.Guard(func() error { var cause error; op, cause = r.Tools.Prepare(ctx, r.Settings, call); return cause })
	if err != nil {
		outcome.Failure = faults.FailureOf(err)

		return nil, outcome, err
	}
	proposal := op.Approval()
	proposal.Required = proposal.Required || proposal.UserRequired
	if proposal.Candidate != nil {
		if err := proposal.Candidate.Check(call); err != nil {
			toolapi.Release(op)
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

func (s *execution) approveExecution(ctx context.Context, ready *readyExecution, approve Approver, emit func(runrecord.Event) error) (result *readyExecution, outcome toolapi.Result, resultErr error) {
	defer func() {
		if resultErr != nil {
			toolapi.Release(ready.operation)
		}
	}()
	call, proposal := ready.call, ready.proposal
	instructionRevision := s.instructionRevision
	outcome = toolapi.Result{Target: proposal.Target}
	var err error
	if err := s.checkTimeBudget(ctx); err != nil {
		return nil, outcome, err
	}
	ctx, cancel := context.WithDeadline(ctx, s.usage.deadline)
	defer cancel()
	approved := !proposal.Required
	waitedForUser := false
	if proposal.Required {
		source := "user"
		if s.runner.Settings.Permission == policy.PermissionAllMighty && !proposal.UserRequired {
			approved, source = true, "all_mighty"
		} else {
			waitedForUser = true
			started := time.Now()
			diagnostics.Log(ctx, diagcatalog.ApprovalWaiting, "call_id", call.ID, "operation", call.Name)
			approved, err = approve(ctx, proposal)
			diagnostics.Log(ctx, diagcatalog.ApprovalDecided, "call_id", call.ID, "duration_ms", time.Since(started).Milliseconds(), "error_code", diagnostics.Code(err), "status", approved)
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
	if waitedForUser {
		if err := s.checkSteering(ctx, instructionRevision); err != nil {
			return nil, outcome, err
		}
	}

	ready.proposal, ready.approved = proposal, approved
	return ready, outcome, nil
}
