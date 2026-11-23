// Purpose: Verify tool attempts remain observations while model requests retain their budget.
// Pattern: Budget contract tests.
// Usage: go test ./internal/agent/run -run TestRequestBudget.
// Related: internal/agent/run/budget_usage.go.
package run

import (
	"context"
	"errors"
	"testing"
)

func TestRequestBudget(t *testing.T) {
	s := &execution{limits: Limits{MaxModelCalls: 200}}
	s.usage.recordTools(1000)
	s.usage.recordTools(1)
	for range 200 {
		if err := s.modelCall(t.Context()); err != nil {
			t.Fatal("model rejected before budget")
		}
	}
	if !errors.Is(s.modelCall(t.Context()), errModelBudget) || s.usage.ToolCalls != 1001 || s.usage.ModelCalls != 200 {
		t.Fatal("request budget overrun")
	}
}

func TestSharedAdmissionCancellationDoesNotConsumeLocalCall(t *testing.T) {
	s := &execution{limits: Limits{MaxModelCalls: 1}, runner: Runner{
		AdmitModel: func(ctx context.Context) (bool, error) { return false, ctx.Err() },
	}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := s.modelCall(ctx); !errors.Is(err, context.Canceled) || s.usage.ModelCalls != 0 {
		t.Fatalf("cancelled admission consumed a call: %v", err)
	}
}
