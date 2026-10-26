// Purpose: Cache input wrapping and expose a shared display/source mapping for cursor and selection.
// Pattern: Read-only presentation projection.
// Usage: Session prepares Window once and passes it to rendering and mouse handlers.
// Related: internal/tui/input/editor.go, internal/tui/textlayout/wrap.go, internal/tui/view/input.go.
package input

import (
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"sort"
	"unicode/utf8"
)

type editorLayout struct {
	cursor, row, column int
	positioned          bool
	source              string
	width               int
	rows                []textlayout.Row
	runeStarts          []int
}
type Window struct {
	Source             string
	Rows               []textlayout.Row
	CursorX, CursorRow int
	runeStarts         []int
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
	starts := make([]int, len(rows))
	position, previous := 0, 0
	for i, row := range rows {
		position += utf8.RuneCountInString(source[previous:row.Start])
		starts[i], previous = position, row.Start
	}
	e.layout = editorLayout{source: source, width: width, rows: rows, runeStarts: starts}
}
func (e *Editor) position() (int, int) {
	if e.layout.positioned && e.layout.cursor == e.Cursor {
		return e.layout.row, e.layout.column
	}
	position := e.Cursor + 2
	index := max(0, sort.Search(len(e.layout.runeStarts), func(i int) bool { return e.layout.runeStarts[i] > position })-1)
	row := e.layout.rows[index]
	offset := row.Start
	for left := position - e.layout.runeStarts[index]; left > 0 && offset < len(e.layout.source); left-- {
		_, size := utf8.DecodeRuneInString(e.layout.source[offset:])
		offset += size
	}
	e.layout.cursor, e.layout.row, e.layout.column, e.layout.positioned = e.Cursor, index, row.Column(e.layout.source, offset), true
	return e.layout.row, e.layout.column
}
func (e *Editor) Window(width, maxRows int) Window {
	e.prepareLayout(width)
	row, x := e.position()
	first := max(0, row-max(1, maxRows)+1)
	end := min(len(e.layout.rows), first+max(1, maxRows))
	return Window{Source: e.layout.source, Rows: e.layout.rows[first:end], CursorX: x, CursorRow: row - first, runeStarts: e.layout.runeStarts[first:end]}
}

func (w Window) Offset(x, row int, after bool) int {
	row = max(0, min(len(w.Rows)-1, row))
	if len(w.Rows) == 0 {
		return 0
	}
	offset := w.Rows[row].Offset(w.Source, x, after)
	if len(w.runeStarts) == len(w.Rows) {
		return max(0, w.runeStarts[row]+utf8.RuneCountInString(w.Source[w.Rows[row].Start:offset])-2)
	}
	return max(0, utf8.RuneCountInString(w.Source[:offset])-2)
}
