// Purpose: Measure unchanged visible-window reuse over a larger indexed transcript.
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
	var transcript Transcript
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		transcript.Prepare(80, v.Blocks, nil)
		_ = transcript.Window(max(0, transcript.Count()-30), 30)
	}
}

func BenchmarkMarkdownHistoryLines(b *testing.B) {
	v := View{Follow: true}
	for range 200 {
		v.AppendBlock(Block{Text: "## Result\n\n**Complete**: 한글 😀 and `code`.\n\n- First item\n- Second item\n", Markdown: true})
	}
	var transcript Transcript
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		transcript.Prepare(80, v.Blocks, nil)
		_ = transcript.Window(max(0, transcript.Count()-30), 30)
	}
}
