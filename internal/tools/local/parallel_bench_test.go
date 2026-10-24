// Purpose: Measure planning sixteen independent file operations without repeating parsing and filesystem reads.
// Pattern: Synthetic filesystem scheduling benchmark.
// Usage: go test ./internal/tools/local -run '^$' -bench BenchmarkParallelPlanning -benchmem.
// Related: internal/tools/local/parallel.go, internal/agent/run/batch.go.
package local

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func BenchmarkParallelPlanning(b *testing.B) {
	root, home := b.TempDir(), b.TempDir()
	e, err := NewExecutor(root, home, filepath.Join(home, ".meca"), func(s string) string { return s })
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { e.Close() })
	var calls []toolapi.Call
	for i := range 16 {
		calls = append(calls, call("write_file", fmt.Sprintf(`{"path":"file%d","content":"body"}`, i)))
	}
	b.ReportAllocs()
	for b.Loop() {
		if end, err := e.ParallelBatch(context.Background(), policy.Settings{Scope: policy.System}, calls, 16); err != nil || end != len(calls) {
			b.Fatalf("planned=%d: %v", end, err)
		}
	}
}
