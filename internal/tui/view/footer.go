// Purpose: Render session status above context-sensitive keyboard hints in two fixed bottom rows.
// Pattern: Stateless terminal renderer.
// Usage: The session reserves FooterRows and supplies current display values to Footer.
// Related: internal/tui/view/header.go, internal/tui/view/keymap.go, internal/tui/session/render.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"slices"
)

const FooterRows = 2

func Footer(screen tcell.Screen, status Status, shortcuts []Shortcut, y, width int) {
	statusLine(screen, status, y, width)
	if !slices.ContainsFunc(shortcuts, func(key Shortcut) bool { return key.Key == "F1" }) {
		shortcuts = append(slices.Clone(shortcuts), Shortcut{Key: "F1", Label: "키 안내"})
	}
	Keymap(screen, shortcuts, y+1, width, FooterRows-1)
}
