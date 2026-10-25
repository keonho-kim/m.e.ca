// Purpose: Verify measured charge validation, optional zeros and independent usage snapshots.
// Pattern: Pure domain contract tests.
// Usage: go test ./internal/agent/model -run TestUsage.
// Related: internal/agent/model/usage.go.
package model

import (
	"math"
	"testing"
)

func TestUsageCostSnapshotsAndTotals(t *testing.T) {
	original := TokenUsage{Input: 10, Output: 2, Cost: new(0.002)}
	clone := original.Clone()
	total := original.Add(TokenUsage{Input: 20, Output: 3, Cost: new(0.003)})
	*original.Cost = 9
	if *clone.Cost != 0.002 || math.Abs(*total.Cost-0.005) > 1e-12 || total.Input != 30 || total.Output != 5 {
		t.Fatalf("snapshots aliased or totals lost: %+v %+v", clone, total)
	}
	if (TokenUsage{}).Add(TokenUsage{}).Cost != nil {
		t.Fatal("invented an unreported charge")
	}
	zero := (TokenUsage{}).Add(TokenUsage{Cost: new(0.0)})
	if zero.Cost == nil || *zero.Cost != 0 {
		t.Fatal("reported free call became unavailable")
	}
}

func TestUsageRejectsInvalidCosts(t *testing.T) {
	for _, cost := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		usage := TokenUsage{Cost: &cost}
		if usage.Check() == nil || usage.CheckTotals() == nil {
			t.Fatalf("invalid charge accepted: %v", cost)
		}
	}
	if err := (TokenUsage{Input: MaxCallTokens + 1}).CheckTotals(); err != nil {
		t.Fatal("request total used the per-call token bound")
	}
}
