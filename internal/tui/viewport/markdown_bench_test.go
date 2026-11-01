// Purpose: Measure mutable paragraphs, fenced code and lists under the live display workload.
// Pattern: Shape-specific streaming benchmark.
// Usage: Run BenchmarkLiveMarkdownShape with -benchmem.
// Related: internal/tui/viewport/block_layout.go, internal/tui/markdown/stream.go.
package viewport

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkLiveMarkdownShape(b *testing.B) {
	for _, size := range []int{32768, 131072} {
		for _, shape := range []string{"paragraph", "code", "list"} {
			b.Run(fmt.Sprintf("%s_%d", shape, size), func(b *testing.B) {
				text := strings.Repeat("plain text for a reasoning paragraph ", size/37)
				if shape == "code" {
					text = "```\n" + strings.Repeat("sample output line\n", size/19)
				}
				if shape == "list" {
					text = strings.Repeat("- **Result**: value of the observation.\n", size/40)
				}
				values := []string{text + "0", text + "1"}
				block := Block{Markdown: true, live: true}
				i := 0
				b.ReportAllocs()
				for b.Loop() {
					block.Text = values[i%2]
					count := block.Count(80)
					block.Window(80, max(0, count-30), 30)
					i++
				}
			})
		}
	}
}
