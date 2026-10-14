// Purpose: Define stored Plan state and validate its identity and approval invariants.
// Pattern: State Machine contracts.
// Usage: Used by Plan transitions and persistence boundaries.
// Related: internal/agent/plan/transitions.go, internal/agent/plan/proposal.go, internal/storage/sqlite/plans.go.
package plan

import (
	"fmt"
	"time"
)

type PlanStatus string

const (
	Planning         PlanStatus = "planning"
	AwaitingApproval PlanStatus = "awaiting_approval"
	PlanCancelled    PlanStatus = "cancelled"
	Approved         PlanStatus = "approved"
	PlanRunning      PlanStatus = "running"
	PlanSucceeded    PlanStatus = "succeeded"
	PlanFailed       PlanStatus = "failed"
	PlanInterrupted  PlanStatus = "interrupted"
)

type Plan struct {
	ID         string     `json:"id"`
	SessionID  string     `json:"session_id"`
	Revision   int        `json:"revision"`
	Status     PlanStatus `json:"status"`
	Objective  string     `json:"objective"`
	Steps      []PlanStep `json:"steps"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ApprovedAt *time.Time `json:"approved_at"`
}

type PlanStep struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func (plan Plan) Check() error {
	if plan.ID == "" || plan.SessionID == "" || plan.Revision < 1 || len(plan.ID) > 128 || len(plan.SessionID) > 128 {
		return fmt.Errorf("invalid stored Plan identity")
	}
	steps := make([]string, 0, len(plan.Steps))
	seen := make(map[string]bool)
	for _, step := range plan.Steps {
		if step.ID == "" || seen[step.ID] {
			return fmt.Errorf("invalid or duplicate Plan step ID")
		}
		seen[step.ID] = true
		switch step.Status {
		case "pending", "running", "succeeded", "failed", "interrupted":
		default:
			return fmt.Errorf("invalid Plan step status")
		}
		steps = append(steps, step.Description)
	}
	if err := checkProposal(PlanProposal{Objective: plan.Objective, Steps: steps}); err != nil {
		return err
	}
	switch plan.Status {
	case Planning, AwaitingApproval, PlanCancelled:
		if plan.ApprovedAt != nil {
			return fmt.Errorf("unapproved Plan contains approval")
		}
	case Approved, PlanRunning, PlanSucceeded, PlanFailed, PlanInterrupted:
		if plan.ApprovedAt == nil {
			return fmt.Errorf("approved Plan is missing approval")
		}
	default:
		return fmt.Errorf("invalid Plan status")
	}
	return nil
}
