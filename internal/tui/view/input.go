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

func Input(screen tcell.Screen, editor input.Editor, y, width, maxRows int) int {
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
	offset := max(0, cursorRow-maxRows+1)
	count := min(maxRows, max(1, len(lines)-offset))
	for row := 0; row < count; row++ {
		if offset+row < len(lines) {
			Text(screen, 0, y+row, width, lines[offset+row], tcell.StyleDefault)
		}
	}
	screen.ShowCursor(cursorX, y+min(count-1, max(0, cursorRow-offset)))
	return count
}
