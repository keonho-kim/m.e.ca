// Purpose: Verify actual charge formatting distinguishes missing, free and very small amounts.
// Pattern: Pure presentation tests.
// Usage: go test ./internal/tui/locale -run TestCost.
// Related: internal/tui/locale/cost.go.
package locale

import "testing"

func TestCostLabelsAndPrecision(t *testing.T) {
	for _, tc := range []struct {
		cost *float64
		want string
	}{
		{nil, "—"}, {new(0.0), "$0"}, {new(0.012345), "$0.012345"}, {new(0.00000001), "<$0.000001"},
	} {
		if got := English.Cost(tc.cost); got != "TOTAL COST "+tc.want {
			t.Fatal(got)
		}
		if got := Korean.Cost(tc.cost); got != "총 비용 "+tc.want {
			t.Fatal(got)
		}
	}
}
