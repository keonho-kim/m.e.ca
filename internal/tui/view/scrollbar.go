// Purpose: Draw a foreground-only scrollbar in the transcript's reserved right column.
// Pattern: Stateless terminal renderer.
// Usage: Chat passes shared viewport geometry after drawing transcript text.
// Related: internal/tui/view/chat.go, internal/tui/viewport/scrollbar.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func Scrollbar(screen tcell.Screen, x, y int, bar viewport.Scrollbar) {
	if !bar.Visible() {
		return
	}
	for row := 0; row < bar.Height; row++ {
		glyph, style := "│", tcell.StyleDefault.Dim(true)
		if row >= bar.ThumbTop && row < bar.ThumbTop+bar.ThumbRows {
			glyph, style = "┃", tcell.StyleDefault.Bold(true)
		}
		screen.Put(x, y+row, glyph, style)
	}
}
