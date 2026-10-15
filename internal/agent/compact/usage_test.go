// Purpose: Verify the 200k default and UTF-8 byte fallback used only for compaction.
// Pattern: Usage contract tests.
// Usage: go test ./internal/agent/compact -run TestRequestUsage.
// Related: internal/agent/compact/usage.go.
package compact

import "testing"

func TestRequestUsageThresholdAndByteFallback(t *testing.T) {
	if Defaults().ThresholdTokens != 200000 {
		t.Fatal("wrong threshold")
	}
	for _, tc := range []struct{ bytes, tokens int64 }{{0, 0}, {1, 1}, {3, 1}, {4, 2}, {600000, 200000}} {
		count := EstimateBytes(tc.bytes)
		if count.Amount != tc.tokens || !count.Estimated || count.Kind != "request" {
			t.Fatalf("%d: %+v", tc.bytes, count)
		}
	}
}
