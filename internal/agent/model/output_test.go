// Purpose: Verify automatic and explicit output budgets respect serving capacity.
// Pattern: Table-driven contract tests.
// Usage: go test ./internal/agent/model -run TestOutputBudget.
// Related: internal/agent/model/output.go.
package model

import "testing"

func TestOutputBudget(t *testing.T) {
	for _, tc := range []struct {
		name                                     string
		requested, capacity, server, input, want int
	}{
		{"automatic", 0, 262144, 0, 0, 196608},
		{"rounded down", 0, 1025, 0, 0, 768},
		{"server output", 0, 262144, 32768, 0, 32768},
		{"remaining input", 0, 262144, 0, 200000, 62144},
		{"manual", 12800, 262144, 0, 0, 12800},
		{"large manual", 100000, 262144, 0, 0, 100000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := OutputBudget(tc.requested, tc.capacity, tc.server, tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("got %d, %v; want %d", got, err, tc.want)
			}
		})
	}
	if _, err := OutputBudget(0, 1024, 0, 1024); err == nil {
		t.Fatal("exhausted context accepted")
	}
}
