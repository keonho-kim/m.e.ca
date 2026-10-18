// Purpose: Show every active keyboard action when the fixed footer cannot fit them.
// Pattern: Stateless terminal renderer.
// Usage: F1 opens session keyboard help; Esc or F1 returns to the previous focus.
// Related: internal/tui/view/keymap.go, internal/tui/session/render.go.
package view

import "github.com/gdamore/tcell/v3"

func KeyboardHelp(screen tcell.Screen, keys []Shortcut, width, height, offset int) {
	Text(screen, 0, 0, width, "현재 화면의 키 안내", tcell.StyleDefault.Bold(true))
	for i := offset; i < len(keys) && i-offset < height-2; i++ {
		Text(screen, 0, 2+i-offset, width, keys[i].Key+"  "+keys[i].Label, tcell.StyleDefault)
	}
}
