// Purpose: Verify tool attempts remain observations while model requests retain their budget.
// Pattern: Budget contract tests.
// Usage: go test ./internal/agent/run -run TestRequestBudget.
// Related: internal/agent/run/budget_usage.go.
package run

import "testing"

func TestRequestBudget(t *testing.T) {
	u := &requestBudget{}
	u.recordTools(1000)
	u.recordTools(1)
	for range 200 {
		if !u.model(200) {
			t.Fatal("model rejected before budget")
		}
	}
	if u.model(200) || u.snapshot().ToolCalls != 1001 || u.snapshot().ModelCalls != 200 {
		t.Fatal("request budget overrun")
	}
}
