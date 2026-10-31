// Purpose: Bound retained Markdown caches without truncating visible text or live prefixes.
// Pattern: Cache ownership regression tests.
// Usage: go test ./internal/tui/viewport -run TestLargeMarkdown.
// Related: internal/tui/viewport/block_layout.go, internal/tui/markdown/stream.go.
package viewport

import (
	"strings"
	"testing"
)

func TestLargeMarkdownDropsDerivedCacheAfterWindow(t *testing.T) {
	text := strings.Repeat("**item** `value`\n\n", 60000) + "last value"
	block := Block{Text: text, Markdown: true}
	count := block.Count(80)
	rows := block.Window(80, count-3, 3)
	if len(rows) == 0 || !strings.Contains(rows[len(rows)-1].Text, "last value") {
		t.Fatal("large Markdown lost visible content")
	}
	if block.layout.parsed != nil || block.layout.metrics != nil || block.stream.RetainedBytes() != 0 {
		t.Fatal("oversized derived cache remained resident")
	}
	if block.Count(80) != count || block.Text != text || len(block.layout.points) == 0 {
		t.Fatal("eviction dropped source or sparse scroll index")
	}
	rows = block.Window(80, 0, 2)
	if len(rows) == 0 || !strings.Contains(rows[0].Text, "item") {
		t.Fatal("evicted Markdown cannot be scrolled back")
	}
}

func TestLargeMarkdownLivePrefixKeepsIncrementalLifetime(t *testing.T) {
	block := Block{Text: "## first\n\ntext\n\n## second\n\nbody\n\n## third\n\nlast", Markdown: true}
	var transcript Transcript
	view := View{Follow: true}
	transcript.Prepare(80, &view, []*Block{&block})
	transcript.Window(0, transcript.Count())
	block.Text += " value"
	transcript.Prepare(80, &view, []*Block{&block})
	transcript.Window(0, transcript.Count())
	if !block.live || block.stream.ReusedRows() == 0 {
		t.Fatal("live prefix was released between deltas")
	}
}
