// Purpose: Verify horizontal workflow separators preserve streaming, wrapping and source coordinates.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/viewport -run TestGroup.
// Related: internal/tui/viewport/group.go, internal/tui/viewport/transcript_index.go.
package viewport

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

func TestGroupJoinsHistoryAndLiveWithoutDuplicatingRules(t *testing.T) {
	v := View{Follow: true}
	f := Group{ID: "review", Title: "Look At This"}
	v.AppendBlock(Block{Text: "Main before"})
	v.AppendBlock(Block{Text: "**Inspect**", Markdown: true, Thinking: true, Group: f})
	live := Block{Text: "결과 output", Markdown: true, Group: f, ID: v.Next + 1}
	var layout Transcript
	layout.Prepare(32, &v, []*Block{&live})
	rows := layout.Window(0, layout.Count())
	var shown strings.Builder
	for _, row := range rows {
		shown.WriteString(row.Text + "\n")
		if textlayout.Width(row.Text) > 32 {
			t.Fatalf("group overflow: %q", row.Text)
		}
		if strings.Contains(row.Text, "Inspect") {
			if !row.Thinking || len(row.Spans) == 0 || row.Copy.Source == "" || row.Copy.Row.Column(row.Copy.Source, row.Copy.Row.Start) != 0 {
				t.Fatalf("group lost styling or copy origin: %+v", row)
			}
		}
	}
	if strings.Count(shown.String(), "Look At This") != 1 || !strings.Contains(shown.String(), "결과 output") {
		t.Fatalf("LAT group split or missing: %s", shown.String())
	}
	// Settling the same live response extends the indexed history prefix.
	v.AppendBlock(live)
	v.AppendBlock(Block{Text: "Main after"})
	layout.Prepare(32, &v, nil)
	rows = layout.Window(0, layout.Count())
	for i := range rows {
		if !strings.Contains(rows[i].Text, "Main after") {
			continue
		}
		if rows[i].Text != "Main after" || rows[i-1].Text != strings.Repeat("-", 32) {
			t.Fatalf("group did not close before main output: %+v", rows)
		}
		layout.Seek(&v, i-2, 2)
		if got := layout.Start(&v, 2); got != i-2 {
			t.Fatalf("grouped scrolling anchor changed: %d", got)
		}
		return
	}
	t.Fatal("main output disappeared")
}

func TestGroupReflowsAndTogglesDetailsAtNarrowWidths(t *testing.T) {
	v := View{Follow: true}
	v.AppendBlock(Block{Text: "unused", Summary: "short", Details: strings.Repeat("한글 table ", 30), Group: Group{ID: "plan", Title: "Look At This"}})
	var layout Transcript
	for _, width := range []int{50, 20, 6, 1, 50} {
		layout.Prepare(width, &v, nil)
		for _, row := range layout.Window(0, layout.Count()) {
			if textlayout.Width(row.Text) > width {
				t.Fatalf("width %d overflow: %q", width, row.Text)
			}
		}
	}
	v.ToggleDetails()
	layout.Prepare(50, &v, nil)
	rows := layout.Window(0, layout.Count())
	if len(rows) != 3 || !strings.Contains(rows[1].Text, "short") {
		t.Fatalf("collapsed grouped output changed: %+v", rows)
	}
}
