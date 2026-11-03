// Purpose: Verify direct row appends preserve caller-owned rows, diff styling and independent snapshots.
// Pattern: Display buffer ownership regression tests.
// Usage: go test ./internal/tui/viewport -run TestAppendWindow.
// Related: internal/tui/viewport/block_window.go, internal/tui/viewport/window.go, internal/tui/viewport/changes.go.
package viewport

import (
	"reflect"
	"testing"
)

func TestAppendWindowPreservesExistingRows(t *testing.T) {
	for _, block := range []Block{
		{Text: "**body**", Markdown: true, Centered: true, SpaceAfter: 2},
		{Text: "+added", Changes: &Changes{Source: "+added", Ranges: []ChangeRange{{Start: 0, End: 6, Kind: '+'}}}, SpaceAfter: 2},
	} {
		want := block.Window(20, 0, block.Count(20))
		rows := make([]Line, 1, 10)
		prefix := Line{Text: "caller-owned", ID: 55, Change: '-'}
		rows[0] = prefix
		got := block.appendWindow(rows, 20, 0, block.Count(20))
		if &got[0] != &rows[0] || !reflect.DeepEqual(got[0], prefix) || !reflect.DeepEqual(got[1:], want) {
			t.Fatalf("append changed ownership, a prefix row or source styles: %+v", got)
		}
		if after := block.appendWindow(got, 20, block.Count(20), 5); len(after) != len(got) || &after[0] != &got[0] {
			t.Fatal("empty source appended rows or replaced caller storage")
		}
	}
}

func TestAppendWindowLeavesPreviousSnapshotsIndependent(t *testing.T) {
	block := Block{Text: "first", SpaceAfter: 2}
	before := block.Window(20, 0, block.Count(20))
	block.Text = "second"
	after := block.Window(20, 0, block.Count(20))
	if before[0].Text != "first" || after[0].Text != "second" || &before[0] == &after[0] {
		t.Fatal("independent windows reused mutable caller-visible storage")
	}
}
