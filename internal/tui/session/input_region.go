// Purpose: Bind input rows and reflowed selections to their editable source and clipboard projection.
// Pattern: Session-owned presentation mapping.
// Usage: Renderers publish regions; mouse selection snapshots them before any edit.
// Related: internal/tui/session/render.go, internal/tui/session/text_selection.go, internal/tui/session/keymap.go, internal/tui/input/layout.go, internal/tui/viewport/viewport.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/tui/input"
	"github.com/keonho-kim/v-code/internal/tui/textlayout"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

type inputRegion struct {
	editor   *input.Editor
	window   input.Window
	y, width int
	revision uint64
}

func (r inputRegion) contains(x, y int) bool {
	return r.editor != nil && x >= 0 && x < r.width && y >= r.y && y < r.y+len(r.window.Rows)
}

func (a *app) captureChatSources(lines []viewport.Line, start, count int) {
	a.chatVisible = lines[start:min(len(lines), start+count)]
	for i := start; i < len(lines) && i-start < count; i++ {
		a.selectionSources[1+i-start] = lines[i].Copy
	}
}

func (a *app) captureInputSources() {
	r := a.inputRegion
	if r.editor == nil {
		return
	}
	for i, row := range r.window.Rows {
		if row.Start < 2 {
			row.Start = min(2, row.End)
			row.Prefix = "> "
		}
		a.selectionSources[r.y+i] = textlayout.Fragment{Source: r.window.Source, Row: row}
	}
}

func (a *app) reanchorInputSelection(body int) {
	if !a.textSelection.Active() || a.textSelection.Dragging() {
		return
	}
	old, current := a.selectionInput, a.inputRegion
	x1, y1, x2, y2, ok := a.textSelection.Bounds()
	if !ok || !old.contains(x1, y1) || !old.contains(x2, y2) || current.editor != old.editor ||
		current.revision != old.revision || current.width != old.width || current.window.Source != old.window.Source {
		return
	}
	if current.y == old.y && len(current.window.Rows) == len(old.window.Rows) && current.window.Rows[0].Start == old.window.Rows[0].Start {
		return
	}
	start := old.window.Rows[y1-old.y].Offset(old.window.Source, x1, false)
	end := old.window.Rows[y2-old.y].Offset(old.window.Source, x2, false)
	x1, y1, first := current.position(start)
	x2, y2, last := current.position(end)
	if !first || !last {
		return
	}
	if x1 == x2 && y1 == y2 {
		_, _, width := a.screen.Get(x2, y2)
		x2 += max(1, width) - 1 // Preserve a selected wide grapheme after snapping to its first cell.
	}
	a.textSelection.Begin(a.screen, x1, y1, 1, body-1)
	a.textSelection.Move(x2, y2)
	a.textSelection.Release()
	a.textSelection.Sources(a.selectionSources)
	a.selectionInput = current
}

func (r inputRegion) position(offset int) (int, int, bool) {
	for i, row := range r.window.Rows {
		if offset >= row.Start && offset <= row.End && !(offset == row.End && row.Soft) {
			return row.Column(r.window.Source, offset), r.y + i, true
		}
	}
	return 0, 0, false
}

func (a *app) clearSelection() {
	a.textSelection.Clear()
	a.selectionInput = inputRegion{}
	a.selectedTranscript = nil
}

func (a *app) selectedInputRange() (int, int, bool) {
	x1, y1, x2, y2, ok := a.textSelection.Bounds()
	r := a.selectionInput
	if !ok || !r.contains(x1, y1) || !r.contains(x2, y2) || r.editor.Revision() != r.revision {
		return 0, 0, false
	}
	start, end := r.window.Offset(x1, y1-r.y, false), r.window.Offset(x2, y2-r.y, true)
	return start, end, start < end
}

func (a *app) deleteSelection() bool {
	start, end, ok := a.selectedInputRange()
	if !ok {
		return false
	}
	a.selectionInput.editor.DeleteRange(start, end)
	a.refreshHints()
	if a.selectionInput.editor == &a.menuInput {
		a.refreshModelHints()
	}
	return true
}
