// Purpose: Render a bounded multiline editor and its visible terminal cursor.
// Pattern: Presentation renderer.
// Usage: Session and file screens supply a read-only editor snapshot and row bounds.
// Related: internal/tui/input/layout.go, internal/tui/view/cells.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/input"
)

func Input(screen tcell.Screen, window input.Window, y, width int) {
	for row, line := range window.Rows {
		Text(screen, 0, y+row, width, line.Text, tcell.StyleDefault)
	}
	screen.ShowCursor(min(width-1, window.CursorX), y+window.CursorRow)
}
