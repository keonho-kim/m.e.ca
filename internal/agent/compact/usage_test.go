// Purpose: Verify the compaction reserve default and UTF-8 byte fallback.
// Pattern: Usage contract tests.
// Usage: go test ./internal/agent/compact -run TestCompactionReserve.
// Related: internal/agent/compact/usage.go.
package compact

import "testing"

func TestCompactionReserveAndByteFallback(t *testing.T) {
	if Defaults().ReserveTokens != 16384 {
		t.Fatal("wrong reserve")
	}
	for _, tc := range []struct{ bytes, tokens int64 }{{0, 0}, {1, 1}, {3, 1}, {4, 2}, {600000, 200000}} {
		count := EstimateBytes(tc.bytes)
		if count.Amount != tc.tokens || !count.Estimated || count.Kind != "request" {
			t.Fatalf("%d: %+v", tc.bytes, count)
		}
	}
}
