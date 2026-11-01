// Purpose: Measure allocation for the read-only phase tool snapshot.
// Pattern: Fixed-catalog microbenchmark.
// Usage: Run BenchmarkReadOnlyDefinitions with -benchmem.
// Related: internal/tools/local/read_only.go, internal/tools/local/catalog/catalog.go.
package local

import "testing"

func BenchmarkReadOnlyDefinitions(b *testing.B) {
	executor := &readOnlyExecutor{}
	b.ReportAllocs()
	for b.Loop() {
		_ = executor.Definitions()
	}
}
