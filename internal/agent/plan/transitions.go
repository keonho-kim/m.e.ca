// Purpose: Apply immutable Plan revision, cancellation, approval and step transitions.
// Pattern: State Machine.
// Usage: Called by Plan creation, approval and execution use cases.
// Related: internal/agent/plan/plan.go, internal/agent/run/plan.go.
package plan

import (
	"fmt"
	"time"
)

func NewPlan(id, session, objective string, steps []string, now time.Time) (Plan, error) {
	if id == "" || session == "" || len(id) > 128 || len(session) > 128 {
		return Plan{}, fmt.Errorf("invalid Plan identifiers")
	}
	proposal := PlanProposal{Objective: objective, Steps: steps}
	if err := checkProposal(proposal); err != nil {
		return Plan{}, err
	}
	plan := Plan{ID: id, SessionID: session, Revision: 1, Status: AwaitingApproval, Objective: objective, CreatedAt: now, UpdatedAt: now}
	for index, description := range steps {
		plan.Steps = append(plan.Steps, PlanStep{ID: fmt.Sprintf("%s:1:%d", id, index+1), Description: description, Status: "pending"})
	}
	return plan, nil
}

func RevisePlan(plan Plan, objective string, steps []string, now time.Time) (Plan, error) {
	if plan.Status != Planning && plan.Status != AwaitingApproval {
		return Plan{}, fmt.Errorf("only unapproved Plans can be revised")
	}
	revised, err := NewPlan(plan.ID, plan.SessionID, objective, steps, now)
	if err != nil {
		return Plan{}, err
	}
	revised.Revision = plan.Revision + 1
	revised.CreatedAt = plan.CreatedAt
	for index := range revised.Steps {
		revised.Steps[index].ID = fmt.Sprintf("%s:%d:%d", plan.ID, revised.Revision, index+1)
	}
	return revised, nil
}

func ApprovePlan(plan Plan, revision int, now time.Time) (Plan, error) {
	if plan.Status != AwaitingApproval || plan.Revision != revision {
		return Plan{}, fmt.Errorf("Plan approval must match the awaiting revision")
	}
	plan.Steps = append([]PlanStep(nil), plan.Steps...)
	plan.Status, plan.ApprovedAt, plan.UpdatedAt = Approved, &now, now
	return plan, nil
}

func CancelPlan(plan Plan, now time.Time) (Plan, error) {
	if plan.Status != Planning && plan.Status != AwaitingApproval {
		return Plan{}, fmt.Errorf("only unapproved active Plans can be cancelled")
	}
	plan.Status, plan.UpdatedAt = PlanCancelled, now
	return plan, plan.Check()
}

func AdvanceStep(plan Plan, index int, status string, now time.Time) (Plan, error) {
	if plan.ApprovedAt == nil || plan.Status != Approved && plan.Status != PlanRunning {
		return Plan{}, fmt.Errorf("approved active Plan required")
	}
	if index < 0 || index >= len(plan.Steps) {
		return Plan{}, fmt.Errorf("invalid Plan step")
	}
	for preceding := 0; preceding < index; preceding++ {
		if plan.Steps[preceding].Status != "succeeded" {
			return Plan{}, fmt.Errorf("preceding Plan steps must finish first")
		}
	}
	previous := plan.Steps[index].Status
	if status == "running" && previous != "pending" || status != "running" && previous != "running" {
		return Plan{}, fmt.Errorf("invalid Plan step transition")
	}
	if status != "running" && status != "succeeded" && status != "failed" && status != "interrupted" {
		return Plan{}, fmt.Errorf("unknown Plan step state")
	}
	plan.Steps = append([]PlanStep(nil), plan.Steps...)
	plan.Steps[index].Status = status
	plan.Status, plan.UpdatedAt = PlanRunning, now
	if status == "failed" {
		plan.Status = PlanFailed
	}
	if status == "interrupted" {
		plan.Status = PlanInterrupted
	}
	if status == "succeeded" && index == len(plan.Steps)-1 {
		plan.Status = PlanSucceeded
	}
	return plan, nil
}
