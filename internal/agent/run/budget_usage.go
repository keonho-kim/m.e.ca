// Purpose: Own action/model counters and the deadline for one serial request.
// Pattern: Request-owned budget state.
// Usage: The run owner counts tool attempts and reserves model calls; file workers never change counters.
// Related: internal/agent/run/runner.go, internal/agent/run/decision_budget.go.
package run

import "time"

type requestBudget struct {
	ToolCalls, ModelCalls int
	deadline              time.Time
}
type usageSnapshot struct {
	ToolCalls, ModelCalls int
	deadline              time.Time
}

func (u *requestBudget) begin(l Limits) {
	if u.deadline.IsZero() {
		u.deadline = time.Now().Add(time.Duration(l.MaxSeconds) * time.Second)
	}
}
func (u *requestBudget) snapshot() usageSnapshot {
	return usageSnapshot{u.ToolCalls, u.ModelCalls, u.deadline}
}
func (u *requestBudget) recordTools(n int) {
	u.ToolCalls += n
}
func (u *requestBudget) model(limit int) bool {
	if u.ModelCalls >= limit {
		return false
	}
	u.ModelCalls++
	return true
}
