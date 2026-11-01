// Purpose: Measure initial Markdown indexing and reflow without warming display caches.
// Pattern: Cold presentation workload benchmark.
// Usage: Run BenchmarkMarkdownInitialLayout with -benchmem.
// Related: internal/tui/viewport/block_layout.go, internal/tui/markdown/render.go.
package viewport

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkMarkdownInitialLayout(b *testing.B) {
	for _, shape := range []string{"paragraphs", "code", "table"} {
		for _, size := range []int{32 << 10, 256 << 10} {
			b.Run(fmt.Sprintf("%s_%d", shape, size), func(b *testing.B) {
				text := strings.Repeat("A paragraph of **observed** work and `source` text.\n\n", size/52)
				if shape == "code" {
					text = "```\n" + strings.Repeat("source content\n", size/15) + "```\n"
				}
				if shape == "table" {
					text = strings.Repeat("| File | Observed change |\n|---|---|\n| src/a | A long description of actual work |\n\n", size/83)
				}
				b.ReportAllocs()
				for b.Loop() {
					block := Block{Text: text, Markdown: true}
					count := block.Count(80)
					if count < 1 || len(block.Window(80, max(0, count-30), 30)) == 0 {
						b.Fatal("empty Markdown layout")
					}
				}
			})
		}
	}
}
