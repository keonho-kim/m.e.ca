// Purpose: Cache input wrapping and expose a shared display/source mapping for cursor and selection.
// Pattern: Read-only presentation projection.
// Usage: Session prepares Window once and passes it to rendering and mouse handlers.
// Related: internal/tui/input/editor.go, internal/tui/textlayout/wrap.go, internal/tui/view/input.go.
package input

import (
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"unicode/utf8"
)

type editorLayout struct {
	cursor, row, column int
	positioned          bool
	source              string
	width               int
	rows                []textlayout.Row
}
type Window struct {
	Source             string
	Rows               []textlayout.Row
	CursorX, CursorRow int
}

func (e *Editor) prepareLayout(width int) {
	width = max(2, width)
	if e.layout.width == width && e.layout.rows != nil {
		return
	}
	source := "> " + e.String()
	rows := textlayout.Wrap(source, width)
	last := &rows[len(rows)-1]
	if last.Column(source, last.End) >= width {
		last.Soft = true
		rows = append(rows, textlayout.Row{Start: len(source), End: len(source)})
	}
	e.layout = editorLayout{source: source, width: width, rows: rows}
}
func (e *Editor) position() (int, int) {
	if e.layout.positioned && e.layout.cursor == e.Cursor {
		return e.layout.row, e.layout.column
	}
	offset := 2 + len(e.Before())
	for i, row := range e.layout.rows {
		if offset <= row.End && !(offset == row.End && row.Soft) {
			e.layout.cursor, e.layout.row, e.layout.column, e.layout.positioned = e.Cursor, i, row.Column(e.layout.source, offset), true
			return e.layout.row, e.layout.column
		}
	}
	return len(e.layout.rows) - 1, 0
}
func (e *Editor) Window(width, maxRows int) Window {
	e.prepareLayout(width)
	row, x := e.position()
	first := max(0, row-max(1, maxRows)+1)
	end := min(len(e.layout.rows), first+max(1, maxRows))
	return Window{Source: e.layout.source, Rows: e.layout.rows[first:end], CursorX: x, CursorRow: row - first}
}

func (w Window) Offset(x, row int, after bool) int {
	row = max(0, min(len(w.Rows)-1, row))
	if len(w.Rows) == 0 {
		return 0
	}
	offset := w.Rows[row].Offset(w.Source, x, after)
	return max(0, utf8.RuneCountInString(w.Source[:offset])-2)
}
