// Purpose: Render a bounded multiline editor and its visible terminal cursor.
// Pattern: Presentation renderer.
// Usage: Session and file screens supply a read-only editor snapshot and row bounds.
// Related: internal/tui/input/editor.go, internal/tui/viewport/viewport.go, internal/tui/view/cells.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/input"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
	"github.com/rivo/uniseg"
)

// InputRows lets the session reserve exactly the visible editor height before drawing.
func InputRows(editor input.Editor, width, maxRows int) int {
	lines, _, _ := inputWindow(editor, width, maxRows)
	return len(lines)
}

func Input(screen tcell.Screen, editor input.Editor, y, width, maxRows int) int {
	lines, cursorX, cursorRow := inputWindow(editor, width, maxRows)
	for row, line := range lines {
		Text(screen, 0, y+row, width, line, tcell.StyleDefault)
	}
	screen.ShowCursor(cursorX, y+cursorRow)
	return len(lines)
}

func inputWindow(editor input.Editor, width, maxRows int) ([]string, int, int) {
	value := editor.String()
	lines := viewport.Wrap("> "+value, width)
	before := "> " + string(editor.Text[:editor.Cursor])
	prior := viewport.Wrap(before, width)
	cursorRow := len(prior) - 1
	cursorX := uniseg.StringWidth(prior[cursorRow])
	if cursorX >= width {
		cursorX = 0
		cursorRow++
	}
	// A cursor exactly after a full-width line needs its own visible row.
	if cursorRow >= len(lines) {
		lines = append(lines, "")
	}
	offset := max(0, cursorRow-maxRows+1)
	count := min(maxRows, max(1, len(lines)-offset))
	return lines[offset : offset+count], cursorX, min(count-1, max(0, cursorRow-offset))
}
