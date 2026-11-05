// Purpose: Verify resident-memory display units and the unavailable marker.
// Pattern: Pure presentation contract tests.
// Usage: go test ./internal/tui/view -run TestMemorySize.
// Related: internal/tui/view/memory.go, internal/tui/session/memory_usage.go.
package view

import "testing"

func TestMemorySize(t *testing.T) {
	for _, tc := range []struct {
		bytes uint64
		want  string
	}{
		{0, "MEM. USAGE —"},
		{1, "MEM. USAGE 0.0 MiB"},
		{1 << 20, "MEM. USAGE 1.0 MiB"},
		{42 << 20, "MEM. USAGE 42.0 MiB"},
		{1 << 30, "MEM. USAGE 1.00 GiB"},
		{3 << 29, "MEM. USAGE 1.50 GiB"},
		{(1 << 30) + (16 << 20), "MEM. USAGE 1.02 GiB"},
		{^uint64(0), "MEM. USAGE 17179869184.00 GiB"},
	} {
		if got := ResidentMemory(tc.bytes).Label(); got != tc.want {
			t.Fatalf("bytes=%d label=%q want=%q", tc.bytes, got, tc.want)
		}
	}
}
