//go:build linux || darwin

// Purpose: Verify and measure an actual current-process resident-memory query.
// Pattern: Native OS boundary test and benchmark.
// Usage: go test ./internal/processmemory; supported hosts run the native implementation.
// Related: internal/processmemory/resident_linux.go, internal/processmemory/resident_darwin.go.
package processmemory

import "testing"

func TestResidentBytes(t *testing.T) {
	value, err := ResidentBytes()
	if err != nil || value == 0 {
		t.Fatalf("current-process RSS=%d error=%v", value, err)
	}
}

func BenchmarkResidentBytes(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ResidentBytes(); err != nil {
			b.Fatal(err)
		}
	}
}
