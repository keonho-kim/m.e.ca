// Purpose: Bound derived Markdown memory while preserving source, scrolling and normal incremental reuse.
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

func TestLargeMarkdownSharesOneLiveDisplayBudget(t *testing.T) {
	text := strings.Repeat("- **Result**: observation value.\n", 32768)
	thinking := Block{Text: text, Markdown: true, Thinking: true, ID: 1}
	response := Block{Text: text + "last result", Markdown: true, ID: 2}
	var transcript Transcript
	view := View{Follow: true}
	transcript.Prepare(80, &view, []*Block{&thinking, &response})
	rows := transcript.Window(max(0, transcript.Count()-30), 30)
	if len(rows) == 0 || !strings.Contains(rows[len(rows)-1].Text, "last result") {
		t.Fatal("cache trimming lost the active reply")
	}
	if thinking.cacheBytes()+response.cacheBytes() > displayCacheBudget {
		t.Fatal("live display cache exceeded its shared budget")
	}
	if thinking.Text != text || response.Text != text+"last result" {
		t.Fatal("cache trimming changed source text")
	}
	if rows := transcript.Window(0, 2); len(rows) == 0 || !strings.Contains(rows[0].Text, "Result") {
		t.Fatal("evicted Thinking cannot be read again")
	}
}
