// Purpose: Preserve transcript rows and release offscreen state across incremental index invalidations.
// Pattern: Differential behavior tests against independent complete layout.
// Usage: Viewport tests cover appends, bounded history, prepend, folding, live removal and reflow.
// Related: internal/tui/viewport/transcript_index.go, internal/tui/viewport/reference_test.go.
package viewport

import (
	"fmt"
	"reflect"
	"testing"
)

func TestIncrementalTranscriptMatchesFullLayout(t *testing.T) {
	view := View{Follow: true, Blocks: make([]Block, 0, 1024)}
	var transcript Transcript
	live := Block{ID: 5000, Text: "**Thinking**\n\ncurrent", Markdown: true, Thinking: true}
	for step := range 1100 {
		view.AppendBlock(Block{Text: fmt.Sprintf("message %d\n\n", step)})
		if step%101 == 0 {
			view.AppendBlock(Block{Summary: "summary", Details: "detail\nsecond\n"})
			view.ToggleDetails()
		}
		if step%137 == 0 {
			view.Prepend([]Block{{Text: "older\n\n\n"}})
		}
		width := 80
		if step%5 == 0 {
			width = 20
		}
		live.ID = view.Next + 1
		live.Text = fmt.Sprintf("**Thinking**\n\nstep %d", step)
		transcript.Prepare(width, &view, []*Block{&live})
		start := max(0, transcript.Count()-12)
		got := transcript.Window(start, 12)
		// Compose only in the reference; the production path must keep its prefix.
		var expected []Line
		for i := range view.Blocks {
			expected = AppendLines(expected, view.Blocks[i].Lines(width))
		}
		expected = AppendLines(expected, live.Lines(width))
		if transcript.Count() != len(expected) || !reflect.DeepEqual(got, expected[start:]) {
			t.Fatalf("step %d changed transcript rows", step)
		}
	}
	transcript.Prepare(80, &view, nil)
	transcript.Window(max(0, transcript.Count()-12), 12)
	view = View{Follow: true}
	view.Append("replacement history", false)
	transcript.Prepare(80, &view, nil)
	if got := transcript.Window(0, 12); len(got) != 1 || got[0].Text != "replacement history" {
		t.Fatal("replaced history reused old source references")
	}
}
