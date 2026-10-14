// Purpose: Fit session status and context-sensitive keyboard hints into one fixed bottom row.
// Pattern: Stateless terminal renderer.
// Usage: The session reserves one row and supplies current display values to Footer.
// Related: internal/tui/view/header.go, internal/tui/view/keymap.go, internal/tui/session/render.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
)

func Footer(screen tcell.Screen, status Status, shortcuts []Shortcut, y, width int) {
	keys := 0
	for _, shortcut := range shortcuts[:min(2, len(shortcuts))] {
		keys += uniseg.StringWidth(shortcut.Key) + uniseg.StringWidth(shortcut.Label) + 3
	}
	statusWidth := max(0, width-min(keys, width/2)-3)
	x := statusLine(screen, status, y, statusWidth)
	Text(screen, x, y, width-x, " │ ", tcell.StyleDefault.Dim(true))
	x += 3
	for _, line := range keymapLines(shortcuts, width-x, 1) {
		drawShortcuts(screen, line, x, y, width)
	}
}
