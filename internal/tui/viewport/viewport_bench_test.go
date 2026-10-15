// Purpose: Measure whole-history wrapping cost at a fixed terminal width before adding caching.
// Pattern: Performance measurement.
// Usage: go test ./internal/tui/viewport -run '^$' -bench BenchmarkHistoryLines -benchmem.
// Related: internal/tui/viewport/viewport.go, internal/tui/session/render_chat.go.
package viewport

import (
	"strings"
	"testing"
)

func BenchmarkHistoryLines(b *testing.B) {
	v := View{Follow: true}
	for range 200 {
		v.Append(strings.Repeat("terminal history text ", 16), false)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = v.Lines(80)
	}
}

func BenchmarkMarkdownHistoryLines(b *testing.B) {
	v := View{Follow: true}
	for range 200 {
		v.AppendBlock(Block{Text: "## Result\n\n**Complete**: 한글 😀 and `code`.\n\n- First item\n- Second item\n", Markdown: true})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = v.Lines(80)
	}
}
