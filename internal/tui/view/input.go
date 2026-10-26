// Purpose: Render a bounded multiline editor and its visible terminal cursor.
// Pattern: Presentation renderer.
// Usage: Chat and files use Input; setting values use InputAt with their column offset.
// Related: internal/tui/input/layout.go, internal/tui/view/cells.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/input"
)

func Input(screen tcell.Screen, window input.Window, y, width int) {
	InputAt(screen, window, 0, y, width)
}

func InputAt(screen tcell.Screen, window input.Window, x, y, width int) {
	if width <= 0 {
		return
	}
	for row, line := range window.Rows {
		Text(screen, x, y+row, width, line.Text, tcell.StyleDefault)
	}
	screen.ShowCursor(x+min(width-1, window.CursorX), y+window.CursorRow)
}
