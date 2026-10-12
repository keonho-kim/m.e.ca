// Purpose: Own versioned Plans and deterministic approval and step transitions.
// Pattern: State Machine.
// Usage: Used by planning requests, approval handling and SQLite recovery.
// Related: internal/agent/plan/plan_test.go, internal/storage/sqlite/plans.go, SYSTEM.md.

package plan

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type PlanStatus string

const (
	Planning         PlanStatus = "planning"
	AwaitingApproval PlanStatus = "awaiting_approval"
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

type PlanProposal struct {
	Objective string   `json:"objective"`
	Steps     []string `json:"steps"`
}

func ParsePlanProposal(content string) (PlanProposal, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json\n") && strings.HasSuffix(content, "\n```") {
		content = strings.TrimSuffix(strings.TrimPrefix(content, "```json\n"), "\n```")
	}
	if len(content) > 64*1024 {
		return PlanProposal{}, fmt.Errorf("Plan proposal exceeds size limit")
	}
	var proposal PlanProposal
	if err := json.Unmarshal([]byte(content), &proposal); err != nil {
		return proposal, fmt.Errorf("invalid structured Plan: %w", err)
	}
	if err := checkProposal(proposal); err != nil {
		return proposal, err
	}
	return proposal, nil
}

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
	case Planning, AwaitingApproval:
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

func checkProposal(proposal PlanProposal) error {
	if strings.TrimSpace(proposal.Objective) == "" || len(proposal.Objective) > 16*1024 || len(proposal.Steps) == 0 || len(proposal.Steps) > 50 {
		return fmt.Errorf("Plan requires a bounded objective and 1–50 steps")
	}
	for _, description := range proposal.Steps {
		if strings.TrimSpace(description) == "" || len(description) > 4096 {
			return fmt.Errorf("invalid Plan step description")
		}
	}
	return nil
}
