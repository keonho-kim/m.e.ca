// Purpose: Own action/model counters and the shared deadline for one request.
// Pattern: Synchronized request budget.
// Usage: The runtime and its auxiliary model worker reserve calls against one bound.
// Related: internal/agent/run/runner.go, internal/agent/run/decision_budget.go.
package run

import (
	"sync"
	"time"
)

type requestBudget struct {
	mu                    sync.Mutex
	ToolCalls, ModelCalls int
	deadline              time.Time
}
type usageSnapshot struct {
	ToolCalls, ModelCalls int
	deadline              time.Time
}

func (u *requestBudget) begin(l Limits) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.deadline.IsZero() {
		u.deadline = time.Now().Add(time.Duration(l.MaxSeconds) * time.Second)
	}
}
func (u *requestBudget) snapshot() usageSnapshot {
	u.mu.Lock()
	defer u.mu.Unlock()
	return usageSnapshot{u.ToolCalls, u.ModelCalls, u.deadline}
}
func (u *requestBudget) recordTools(n int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.ToolCalls += n
}
func (u *requestBudget) model(limit int) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.ModelCalls >= limit {
		return false
	}
	u.ModelCalls++
	return true
}
