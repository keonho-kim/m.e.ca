// Purpose: Own action/model counters and the deadline for one request.
// Pattern: Serial request-owned budget.
// Usage: The runtime owner counts model and tool calls against one request bound.
// Related: internal/agent/run/runner.go, internal/agent/run/decision_budget.go.
package run

import (
	"time"
)

type requestBudget struct {
	ToolCalls, ModelCalls int
	deadline              time.Time
}

func (u *requestBudget) begin(l Limits) {
	if u.deadline.IsZero() {
		u.deadline = time.Now().Add(time.Duration(l.MaxSeconds) * time.Second)
	}
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
