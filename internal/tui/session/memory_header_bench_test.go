// Purpose: Compare status-only painting with full body rendering for an unchanged transcript.
// Pattern: Virtual terminal workload benchmark.
// Usage: go test ./internal/tui/session -run '^$' -bench BenchmarkMemoryRefresh -benchmem.
// Related: internal/tui/session/render_header.go, internal/tui/session/render.go, internal/tui/session/memory_usage.go.
package session

import (
	"strings"
	"testing"
)

func BenchmarkMemoryRefresh(b *testing.B) {
	for _, full := range []bool{false, true} {
		name := "header"
		if full {
			name = "full"
		}
		b.Run(name, func(b *testing.B) {
			a := screenFixture(b)
			a.chat.output.Append(strings.Repeat("Unchanged transcript line with source text.\n", 512), false)
			a.compose.editor.Set("draft text")
			var page uint64
			a.memory.read = func() (uint64, error) { page ^= 1; return (42 + page) << 20, nil }
			a.memory.sample()
			a.render()
			b.ReportAllocs()
			for b.Loop() {
				a.memory.sample()
				if full {
					a.render()
				} else {
					a.renderHeader()
				}
			}
		})
	}
}
