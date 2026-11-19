// Purpose: Verify the UTF-8 byte fallback for estimating context occupancy.
// Pattern: Usage contract tests.
// Usage: go test ./internal/agent/compact -run TestCompactionByteEstimate.
// Related: internal/agent/compact/usage.go.
package compact

import "testing"

func TestCompactionByteEstimate(t *testing.T) {
	for _, tc := range []struct{ bytes, tokens int64 }{{0, 0}, {1, 1}, {3, 1}, {4, 2}, {600000, 200000}} {
		count := EstimateBytes(tc.bytes)
		if count.Amount != tc.tokens || !count.Estimated || count.Kind != "request" {
			t.Fatalf("%d: %+v", tc.bytes, count)
		}
	}
}
