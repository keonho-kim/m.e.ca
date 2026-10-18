// Purpose: Verify one request cannot reserve actions or model calls beyond its budget.
// Pattern: Budget contract tests.
// Usage: go test ./internal/agent/run -run TestRequestBudget.
// Related: internal/agent/run/budget_usage.go.
package run

import "testing"

func TestRequestBudget(t *testing.T) {
	u := &requestBudget{}
	for range 100 {
		if !u.actions(1, 100) {
			t.Fatal("action rejected before budget")
		}
	}
	for range 200 {
		if !u.model(200) {
			t.Fatal("model rejected before budget")
		}
	}
	if u.actions(1, 100) || u.model(200) || u.snapshot().Iterations != 100 || u.snapshot().ModelCalls != 200 {
		t.Fatal("request budget overrun")
	}
}
