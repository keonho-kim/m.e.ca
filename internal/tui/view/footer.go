// Purpose: Measure and render model status above one or two bottom keyboard-hint rows.
// Pattern: Stateless terminal renderer.
// Usage: The session measures FooterRows and renders Footer inside that reserved region.
// Related: internal/tui/view/header.go, internal/tui/view/keymap.go, internal/tui/session/render.go.
package view

import "github.com/gdamore/tcell/v3"

// FooterRows includes model status and only the keyboard rows that are needed.
func FooterRows(shortcuts []Shortcut, width, height int) int {
	if height <= 0 {
		return 0
	}
	// Keep two body rows available for the activity line and composer.
	keyRows := min(MaxKeymapRows, max(0, height-3))
	return 1 + KeymapRows(shortcuts, width, keyRows)
}

func Footer(screen tcell.Screen, status Status, shortcuts []Shortcut, y, width int) {
	statusLine(screen, status, y, width)
	_, height := screen.Size()
	Keymap(screen, shortcuts, y+1, width, min(MaxKeymapRows, max(0, height-y-1)))
}
