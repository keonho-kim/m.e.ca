// Purpose: Reflow the edited suffix and publish immutable windows of visible input rows.
// Pattern: Read-only presentation projection.
// Usage: Session prepares Window once and passes it to rendering and mouse handlers.
// Related: internal/tui/input/editor.go, internal/tui/textlayout/wrap.go, internal/tui/view/input.go.
package input

import (
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

type editorLayout struct {
	cursor, row, column      int
	positioned               bool
	source                   string
	width                    int
	rows                     []textlayout.Row
	runeStarts               []int
	prefixBytes, prefixRunes int
	restart                  int
	dirty                    bool
	padding                  string
	visible                  Window
	visibleStart             int
}
type Window struct {
	Source             string
	Rows               []textlayout.Row
	CursorX, CursorRow int
	PrefixBytes        int
	runeStarts         []int
	prefixRunes        int
}

// SetPrefix changes display-only annotations without changing the editable body or revision.
// An empty prefix restores the standard prompt; navigation uses the same source projection.
func (e *Editor) SetPrefix(prefix string) {
	if e.prefix == prefix {
		return
	}
	e.prefix = prefix
	e.display = ""
	e.layout = editorLayout{}
	e.vertical = false
}

func (e *Editor) prompt() string {
	if e.prefix != "" {
		return e.prefix
	}
	return "> "
}

func (e *Editor) displaySource() string {
	if e.display == "" {
		prefix := e.prompt()
		e.display = prefix + e.String()
		e.text = e.display[len(prefix):]
		e.textCapacity = len(e.display)
	}
	return e.display
}

func (l *editorLayout) rowAt(position int) int {
	return max(0, sort.Search(len(l.runeStarts), func(i int) bool { return l.runeStarts[i] > position })-1)
}

func (e *Editor) invalidateLayout(start int) {
	l := &e.layout
	if len(l.rows) == 0 {
		return
	}
	position := start + l.prefixRunes
	if l.dirty && position > l.runeStarts[l.restart] {
		return
	}
	// Reflow the previous row as well. An edit can join a cluster or a line
	// across the old wrap boundary, including at the very end of the draft.
	l.restart = max(0, l.rowAt(position)-1)
	l.dirty = true
	l.source = ""
	l.visible = Window{}
	l.positioned = false
}

func (e *Editor) prepareLayout(width int) {
	width = max(2, width)
	l := &e.layout
	if l.width == width && l.rows != nil && !l.dirty {
		return
	}
	prefix := e.prompt()
	source := e.displaySource()
	start, position := 0, 0
	indent := ""
	if l.width == width && l.dirty {
		row := l.rows[l.restart]
		start, position, indent = row.Start, l.runeStarts[l.restart], row.Prefix
		l.rows, l.runeStarts = l.rows[:l.restart], l.runeStarts[:l.restart]
	} else {
		*l = editorLayout{}
	}
	previous := start
	for row := range textlayout.RowsFrom(source, width, start, indent) {
		position += utf8.RuneCountInString(source[previous:row.Start])
		previous = row.Start
		// Indexed rows own offsets and indentation only. A row from the old
		// source must not retain that source after a small edit.
		if len(row.Prefix) > len(l.padding) {
			l.padding = strings.Repeat(" ", len(row.Prefix))
		}
		row.Text, row.Prefix = "", l.padding[:len(row.Prefix)]
		l.rows = append(l.rows, row)
		l.runeStarts = append(l.runeStarts, position)
	}
	last := &l.rows[len(l.rows)-1]
	if last.Column(source, last.End) >= width {
		last.Soft = true
		l.rows = append(l.rows, textlayout.Row{Start: len(source), End: len(source)})
		l.runeStarts = append(l.runeStarts, position+utf8.RuneCountInString(source[previous:]))
	}
	if cap(l.rows) > 4*max(64, len(l.rows)) {
		l.rows, l.runeStarts = slices.Clone(l.rows), slices.Clone(l.runeStarts)
	}
	l.source, l.width = source, width
	l.prefixBytes, l.prefixRunes = len(prefix), utf8.RuneCountInString(prefix)
	l.dirty, l.positioned = false, false
}
func (e *Editor) position() (int, int) {
	if e.layout.positioned && e.layout.cursor == e.Cursor {
		return e.layout.row, e.layout.column
	}
	position := e.Cursor + e.layout.prefixRunes
	index := e.layout.rowAt(position)
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
	l := &e.layout
	if l.visibleStart != first || len(l.visible.Rows) != end-first {
		visible := slices.Clone(l.rows[first:end])
		for i := range visible {
			r := &visible[i]
			for formatted := range textlayout.RowsFrom(l.source[:r.End], l.width, r.Start, r.Prefix) {
				r.Text = formatted.Text
				break
			}
		}
		l.visibleStart = first
		l.visible = Window{Source: l.source, Rows: visible, PrefixBytes: l.prefixBytes, prefixRunes: l.prefixRunes, runeStarts: slices.Clone(l.runeStarts[first:end])}
	}
	value := l.visible
	value.CursorX, value.CursorRow = x, row-first
	return value
}
func (w Window) Offset(x, row int, after bool) int {
	row = max(0, min(len(w.Rows)-1, row))
	if len(w.Rows) == 0 {
		return 0
	}
	offset := w.Rows[row].Offset(w.Source, x, after)
	if len(w.runeStarts) == len(w.Rows) {
		return max(0, w.runeStarts[row]+utf8.RuneCountInString(w.Source[w.Rows[row].Start:offset])-w.prefixRunes)
	}
	return max(0, utf8.RuneCountInString(w.Source[:offset])-w.prefixRunes)
}
