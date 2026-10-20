// Purpose: Render server-default and explicit Thinking choices with the selected value emphasized.
// Pattern: Presentation renderer.
// Usage: The model menu supplies its current candidate and available API controls.
// Related: internal/tui/session/menus.go, internal/tui/session/render.go, internal/tui/view/styles.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/locale"
	"github.com/rivo/uniseg"
)

func Thinking(screen tcell.Screen, y, width int, choice *bool, supported, focused, onOnly bool, lang locale.Language) {
	x := 0
	part := func(text string, style tcell.Style) {
		Text(screen, x, y, max(0, width-x), text, style)
		x += uniseg.StringWidth(text)
	}
	part(lang.Text("Thinking")+" ", SelectionStyle(focused))
	if onOnly {
		part(lang.Text("ON"), SelectionStyle(true))
		part(" ("+lang.Text("OFF unavailable")+")", tcell.StyleDefault.Dim(true))
		return
	}
	part(lang.Text("DEFAULT"), SelectionStyle(choice == nil))
	part(" / ", tcell.StyleDefault.Dim(true))
	part(lang.Text("ON"), SelectionStyle(choice != nil && *choice && supported))
	part(" / ", tcell.StyleDefault.Dim(true))
	part(lang.Text("OFF"), SelectionStyle(supported && choice != nil && !*choice))
	if !supported {
		part(" ("+lang.Text("ON/OFF control unavailable")+")", tcell.StyleDefault.Dim(true))
	}
}
