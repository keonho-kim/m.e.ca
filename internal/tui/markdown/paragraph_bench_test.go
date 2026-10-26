// Purpose: Measure multi-line paragraph assembly without quadratic text concatenation.
// Pattern: Markdown allocation regression benchmark.
// Usage: go test ./internal/tui/markdown -run '^$' -bench BenchmarkParagraph -benchmem.
// Related: internal/tui/markdown/inline.go, internal/tui/markdown/render.go.
package markdown

import (
	"strings"
	"testing"
)

func BenchmarkParagraph(b *testing.B) {
	text := strings.Repeat("A long paragraph continues on the next source line with 한글.\n", 512)
	b.ReportAllocs()
	for b.Loop() {
		Render(text, 80)
	}
}
