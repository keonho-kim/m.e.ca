// Purpose: Render one inline Thinking ON/OFF choice with the selected value emphasized.
// Pattern: Presentation renderer.
// Usage: The model menu supplies its current candidate and verified control support.
// Related: internal/tui/session/menus.go, internal/tui/session/render.go, internal/tui/view/styles.go.
package view

import "github.com/gdamore/tcell/v3"

func Thinking(screen tcell.Screen, y, width int, on, supported, focused bool) {
	Text(screen, 0, y, width, "Thinking ", SelectionStyle(focused))
	Text(screen, 9, y, max(0, width-9), "ON", SelectionStyle(supported && on))
	Text(screen, 11, y, max(0, width-11), " / ", tcell.StyleDefault.Dim(true))
	Text(screen, 14, y, max(0, width-14), "OFF", SelectionStyle(supported && !on))
	if !supported {
		Text(screen, 17, y, max(0, width-17), " (unavailable)", tcell.StyleDefault.Dim(true))
	}
}
