// Purpose: Verify bounded transcript windows preserve full-layout rows, anchors and copy offsets.
// Pattern: Differential presentation tests.
// Usage: go test ./internal/tui/viewport -run TestWindow.
// Related: internal/tui/viewport/window.go, internal/tui/viewport/block_layout.go.
package viewport

import (
	"reflect"
	"strings"
	"testing"
)

func TestWindowMatchesFullComposition(t *testing.T) {
	for _, width := range []int{1, 5, 20, 80} {
		view := View{Follow: true}
		view.Append("first\n\n\n", false)
		view.AppendBlock(Block{Text: "\n\n# Heading\n\n- **한글** 😀\n- second\n\n", Markdown: true})
		view.Append("    long literal text and\ttabs\n\n\nnext", false)
		var expected []Line
		for i := range view.Blocks {
			expected = AppendLines(expected, view.Blocks[i].Lines(width))
		}
		var transcript Transcript
		transcript.Prepare(width, view.Blocks, nil)
		if transcript.Count() != len(expected) {
			t.Fatalf("width %d: count %d want %d", width, transcript.Count(), len(expected))
		}
		for at := range expected {
			want := expected[at:min(at+3, len(expected))]
			if got := transcript.Window(at, 3); !reflect.DeepEqual(got, want) {
				t.Fatalf("width=%d at=%d:\n%+v\nwant %+v", width, at, got, want)
			}
		}
		transcript.Scroll(&view, -4, 3)
		id, row := view.AnchorID, view.AnchorRow
		view.Append("latest", false)
		transcript.Prepare(width, view.Blocks, nil)
		got := transcript.Window(transcript.Start(&view, 3), 1)
		if !view.Follow && (got[0].ID != id || got[0].Row != row) {
			t.Fatal("append moved the reading anchor")
		}
	}
}

func TestWindowLargeLiteralHasSparseLayout(t *testing.T) {
	block := Block{Text: strings.Repeat("line\n", 131072)}
	if got := block.Count(80); got != 131073 {
		t.Fatal(got)
	}
	if len(block.layout.points) > 520 {
		t.Fatalf("retained %d checkpoints for short lines", len(block.layout.points))
	}
	rows := block.Window(80, 131070, 3)
	if len(rows) != 3 || rows[0].Copy.Row.Start != 131070*5 || rows[2].Text != "" {
		t.Fatalf("bad last window: %+v", rows)
	}
}

func TestWindowOversizedMarkdownUsesSparseLiteral(t *testing.T) {
	text := strings.Repeat("x\n", 8*1024*1024+1)
	block := Block{Text: text, Markdown: true}
	count := block.Count(80)
	if count != 8*1024*1024+2 || len(block.layout.parsed) != 0 {
		t.Fatalf("oversized Markdown constructed parsed rows: %d", count)
	}
	rows := block.Window(80, count-2, 2)
	if len(rows) != 2 || rows[0].Text != "x" || rows[1].Text != "" {
		t.Fatalf("literal tail: %+v", rows)
	}
}
