// Purpose: Render the selected model inside the fixed footer.
// Pattern: Presentation renderer.
// Usage: The session renderer supplies current values without sharing mutable task state.
// Related: internal/tui/session/render.go, internal/tui/locale/menus.go, internal/tui/view/cells.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/locale"
)

type Status struct {
	Language locale.Language
	Model    string
}

func statusLine(screen tcell.Screen, status Status, y, w int) {
	model := status.Model
	if model == "" {
		model = status.Language.Text("no-model")
	}
	Text(screen, 0, y, w, model, tcell.StyleDefault)
}
