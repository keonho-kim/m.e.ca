// Purpose: Own action/model counters and the deadline for one serial request.
// Pattern: Request-owned budget state.
// Usage: The run owner reserves calls before dispatch; file workers never change counters.
// Related: internal/agent/run/runner.go, internal/agent/run/decision_budget.go.
package run

import "time"

type requestBudget struct {
	Iterations, ModelCalls int
	deadline               time.Time
}
type usageSnapshot struct {
	Iterations, ModelCalls int
	deadline               time.Time
}

func (u *requestBudget) begin(l Limits) {
	if u.deadline.IsZero() {
		u.deadline = time.Now().Add(time.Duration(l.MaxSeconds) * time.Second)
	}
}
func (u *requestBudget) snapshot() usageSnapshot {
	return usageSnapshot{u.Iterations, u.ModelCalls, u.deadline}
}
func (u *requestBudget) actions(n, limit int) bool {
	if u.Iterations+n > limit {
		return false
	}
	u.Iterations += n
	return true
}
func (u *requestBudget) model(limit int) bool {
	if u.ModelCalls >= limit {
		return false
	}
	u.ModelCalls++
	return true
}
