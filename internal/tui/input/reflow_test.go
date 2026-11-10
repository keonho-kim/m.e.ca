// Purpose: Compare incremental edits against a fresh layout and protect published window snapshots.
// Pattern: Differential terminal layout contracts.
// Usage: go test ./internal/tui/input -run TestEditedLayout.
// Related: internal/tui/input/layout.go, internal/tui/textlayout/wrap.go.
package input

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

func TestEditedLayoutMatchesFreshWrap(t *testing.T) {
	values := []string{"한글", "👩‍💻", "a\u0301", "🇰🇷", "\u200d", "\t", "\n", "    ", "abc"}
	for _, width := range []int{2, 5, 12, 80} {
		random := rand.New(rand.NewPCG(1, uint64(width)))
		var editor Editor
		editor.Set("    abc 한글\n\t👩‍💻 unicode 🇰🇷")
		editor.SetPrefix("> [IMAGE:1] ")
		for step := 0; step < 180; step++ {
			old := editor.Window(width, 5)
			originalRows := slices.Clone(old.Rows)
			for edit := 0; edit < 2; edit++ {
				editor.MoveTo(random.IntN(editor.Len() + 1))
				if random.IntN(2) == 0 {
					editor.Insert(values[random.IntN(len(values))])
				} else {
					editor.Backspace()
				}
			}
			current := editor.Window(width, 5)
			var fresh Editor
			fresh.Set(editor.String())
			fresh.SetPrefix("> [IMAGE:1] ")
			fresh.MoveTo(editor.Cursor)
			want := fresh.Window(width, 5)
			if !reflect.DeepEqual(current, want) {
				t.Fatalf("width %d edit %d: got %+v, fresh %+v", width, step, current, want)
			}
			if !slices.Equal(old.Rows, originalRows) {
				t.Fatal("an edit mutated a published window")
			}
			rows := textlayout.Wrap(current.Source, width)
			last := &rows[len(rows)-1]
			if last.Column(current.Source, last.End) >= width {
				last.Soft = true
				rows = append(rows, textlayout.Row{Start: len(current.Source), End: len(current.Source)})
			}
			index, _ := editor.position()
			start := index - current.CursorRow
			if !slices.Equal(current.Rows, rows[start:start+len(current.Rows)]) {
				t.Fatalf("width %d edit %d: displayed %+v, full wrap %+v", width, step, current.Rows, rows[start:start+len(current.Rows)])
			}
		}
	}
}
