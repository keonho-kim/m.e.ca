// Purpose: Project a horizontally scrolling input field with exact source offsets.
// Pattern: Pure presentation projection.
// Usage: Connection fields share source mapping with chat input and selection.
// Related: internal/tui/input/layout.go, internal/tui/view/input.go.
package input

import (
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"github.com/rivo/uniseg"
	"strings"
)

func (e *Editor) FieldWindow(width int, masked bool) Window {
	value, before := e.String(), e.Before()
	if masked {
		value, before = strings.Repeat("*", len(e.runes)), strings.Repeat("*", e.Cursor)
	}
	source := "> " + value
	x := 2 + uniseg.StringWidth(before)
	row := textlayout.Row{Text: source, End: len(source)}
	if x >= width {
		start := row.Offset(source, x-max(1, width-3), false)
		row = textlayout.Row{Text: "> " + source[start:], Start: start, End: len(source), Prefix: "> "}
		x = row.Column(source, 2+len(before))
	}
	return Window{Source: source, Rows: []textlayout.Row{row}, CursorX: x}
}
