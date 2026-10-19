// Purpose: Render one inline Thinking ON/OFF choice with the selected value emphasized.
// Pattern: Presentation renderer.
// Usage: The model menu supplies its current candidate and verified control support.
// Related: internal/tui/session/menus.go, internal/tui/session/render.go, internal/tui/view/styles.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/locale"
	"github.com/rivo/uniseg"
)

func Thinking(screen tcell.Screen, y, width int, on, supported, focused, onOnly bool, lang locale.Language) {
	x := 0
	part := func(text string, style tcell.Style) {
		Text(screen, x, y, max(0, width-x), text, style)
		x += uniseg.StringWidth(text)
	}
	part(lang.Text("Thinking")+" ", SelectionStyle(focused))
	part(lang.Text("ON"), SelectionStyle(on && supported || onOnly))
	if onOnly {
		part(" ("+lang.Text("OFF unavailable")+")", tcell.StyleDefault.Dim(true))
		return
	}
	part(" / ", tcell.StyleDefault.Dim(true))
	part(lang.Text("OFF"), SelectionStyle(supported && !on))
	if !supported {
		part(" ("+lang.Text("unavailable")+")", tcell.StyleDefault.Dim(true))
	}
}
