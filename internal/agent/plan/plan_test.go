// Purpose: Verify versioned Plan approval and deterministic step transitions.
// Pattern: State-transition tests.
// Usage: go test ./internal/agent/plan.
// Related: internal/agent/plan/plan.go, SYSTEM.md.

package plan

import (
	"testing"
	"time"
)

func TestPlanRevisionAndApproval(t *testing.T) {
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	plan, err := NewPlan("plan_1", "session_1", "Fix auth", []string{"Read auth", "Change auth"}, now)
	if err != nil {
		t.Fatal(err)
	}
	revised, err := RevisePlan(plan, "Fix auth safely", []string{"Read tests", "Change auth", "Run tests"}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if revised.Revision != 2 || plan.Revision != 1 || len(plan.Steps) != 2 || revised.Status != AwaitingApproval {
		t.Fatalf("revision mutated prior plan: %+v, %+v", plan, revised)
	}
	if _, err := ApprovePlan(revised, 1, now); err == nil {
		t.Fatal("approved stale revision")
	}
	approved, err := ApprovePlan(revised, 2, now.Add(2*time.Minute))
	if err != nil || approved.Status != Approved || approved.ApprovedAt == nil {
		t.Fatalf("approval: %+v, %v", approved, err)
	}
	if revised.ApprovedAt != nil {
		t.Fatal("approval mutated prior revision")
	}
	if _, err := RevisePlan(approved, "new objective", []string{"Other"}, now); err == nil {
		t.Fatal("revised approved plan")
	}
	running, err := AdvanceStep(approved, 0, "running", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdvanceStep(running, 1, "running", now); err == nil {
		t.Fatal("skipped causal step ordering")
	}
	completed, err := AdvanceStep(running, 0, "succeeded", now)
	if err != nil || completed.Steps[0].Status != "succeeded" || approved.Steps[0].Status != "pending" {
		t.Fatalf("step: %+v, %v", completed, err)
	}
}

func TestPlanRejectsInvalidStructure(t *testing.T) {
	for _, steps := range [][]string{nil, {""}, {"valid", "  "}} {
		if _, err := NewPlan("plan_1", "session_1", "goal", steps, time.Now()); err == nil {
			t.Errorf("accepted invalid steps %q", steps)
		}
	}
	if _, err := ParsePlanProposal(`{"objective":"goal","steps":["read"]} trailing`); err == nil {
		t.Fatal("accepted trailing output")
	}
	proposal, err := ParsePlanProposal("```json\n{\"objective\":\"goal\",\"steps\":[\"read\"]}\n```")
	if err != nil || len(proposal.Steps) != 1 {
		t.Fatalf("proposal: %+v, %v", proposal, err)
	}
}
