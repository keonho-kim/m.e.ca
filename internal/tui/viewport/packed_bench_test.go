// Purpose: Measure completed Markdown scrolling after the full row cache exceeds its budget.
// Pattern: Fixed-document visible-window benchmark.
// Usage: go test ./internal/tui/viewport -run '^$' -bench BenchmarkCompletedMarkdownScroll -benchmem.
// Related: internal/tui/viewport/markdown_cache.go, internal/tui/markdown/packed.go.
package viewport

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkCompletedMarkdownScroll(b *testing.B) {
	for _, paragraphs := range []int{4000, 60000} {
		b.Run(fmt.Sprint(paragraphs), func(b *testing.B) {
			block := Block{Text: strings.Repeat("**item** `value`\n\n", paragraphs) + "last value", Markdown: true}
			count := block.Count(80)
			block.Window(80, count-30, 30)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if rows := block.Window(80, count-30-i%2, 30); len(rows) != 30 {
					b.Fatal(len(rows))
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(block.cacheBytes()), "retained-B")
		})
	}
}

func BenchmarkCompletedMarkdownInitial(b *testing.B) {
	text := strings.Repeat("**item** `value`\n\n", 60000) + "last value"
	b.ReportAllocs()
	for b.Loop() {
		block := Block{Text: text, Markdown: true}
		count := block.Count(80)
		if rows := block.Window(80, count-30, 30); len(rows) != 30 {
			b.Fatal(len(rows))
		}
	}
}
