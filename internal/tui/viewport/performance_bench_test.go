// Purpose: Measure cold indexed windows and changed Markdown independently of unchanged-window reuse.
// Pattern: Performance regression measurements.
// Usage: go test ./internal/tui/viewport -run '^$' -bench 'Benchmark(Cold|Changed)' -benchmem.
// Related: internal/tui/viewport/viewport.go, internal/tui/markdown/layout.go.
package viewport

import (
	"strconv"
	"strings"
	"testing"
)

func BenchmarkColdLiteral(b *testing.B) {
	text := strings.Repeat("x\n", 131072)
	b.ReportAllocs()
	for b.Loop() {
		block := Block{Text: text}
		block.Window(80, max(0, block.Count(80)-30), 30)
	}
}

func BenchmarkChangedMarkdown(b *testing.B) {
	text := strings.Repeat("- **Result**: example code and `value`.\n", 128*1024/40)
	block := Block{Markdown: true}
	i := 0
	b.ReportAllocs()
	for b.Loop() {
		block.Text = text + strconv.Itoa(i%2)
		block.Window(80, max(0, block.Count(80)-30), 30)
		i++
	}
}
