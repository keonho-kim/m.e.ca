// Purpose: Identify the exact approved Plan step associated with an execution record.
// Pattern: Simple Module.
// Usage: Runtime and compaction attach StepRef to recorded work.
// Related: internal/agent/plan/plan.go.
package plan

type StepRef struct {
	PlanID          string `json:"plan_id"`
	Revision, Index int
	Description     string
	Objective       string
}
