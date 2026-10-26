// Purpose: Move and delete input on grapheme boundaries and visual rows.
// Pattern: Deterministic editor operations.
// Usage: Keyboard and mouse handlers mutate the editor through these methods.
// Related: internal/tui/input/editor.go, internal/tui/input/boundaries.go, internal/tui/input/layout.go.
package input

import "unicode/utf8"

func (e *Editor) Move(n int) {
	e.vertical = false
	for ; n < 0 && e.Cursor > 0; n++ {
		e.Cursor = e.boundary(e.Cursor, false)
	}
	for ; n > 0 && e.Cursor < len(e.runes); n-- {
		e.Cursor = e.boundary(e.Cursor, true)
	}
}

func (e *Editor) MoveTo(position int) {
	e.Cursor = max(0, min(len(e.runes), position))
	e.vertical = false
}
func (e *Editor) Backspace() {
	if e.Cursor > 0 {
		e.replace(e.boundary(e.Cursor, false), e.Cursor, "")
	}
}
func (e *Editor) Delete() {
	if e.Cursor < len(e.runes) {
		e.replace(e.Cursor, e.boundary(e.Cursor, true), "")
	}
}
func (e *Editor) DeleteRange(start, end int) {
	start, end = max(0, start), min(len(e.runes), end)
	if start < end {
		e.replace(start, end, "")
	}
}
func (e *Editor) MoveVertical(delta, width int) {
	e.prepareLayout(width)
	row, x := e.position()
	if !e.vertical {
		e.column = x
		if e.layout.rows[row].Start == 0 {
			e.column -= 2
		}
	}
	row = max(0, min(len(e.layout.rows)-1, row+delta))
	x = e.column
	if e.layout.rows[row].Start == 0 {
		x += 2
	}
	offset := e.layout.rows[row].Offset(e.layout.source, x, false)
	e.Cursor = max(0, e.layout.runeStarts[row]+utf8.RuneCountInString(e.layout.source[e.layout.rows[row].Start:offset])-2)
	e.vertical = true
}
