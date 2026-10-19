// Purpose: Render bounded model and policy values inside the fixed footer.
// Pattern: Presentation renderer.
// Usage: The session renderer supplies current values without sharing mutable task state.
// Related: internal/tui/session/render.go, internal/policy/policy.go, internal/tui/view/cells.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/tui/locale"
	"github.com/rivo/uniseg"
	"strings"
)

type Status struct {
	Language      locale.Language
	Model, Effort string
	Thinking      bool
	Settings      policy.Settings
}

func statusLine(screen tcell.Screen, status Status, y, w int) {
	model, thinking, effort, settings := status.Model, status.Thinking, status.Effort, status.Settings
	if model == "" {
		model = status.Language.Text("no-model")
	}
	suffix := ""
	if thinking {
		suffix = " " + status.Language.Text("Thinking")
		if effort != "" {
			suffix += " " + strings.ToUpper(effort)
		}
	}
	if settings.Permission == policy.AllowAll {
		suffix += " │ " + status.Language.Text("allow-all")
	}
	maxModel := max(0, w-uniseg.StringWidth(suffix))
	Text(screen, 0, y, maxModel, model, tcell.StyleDefault)
	x := min(uniseg.StringWidth(model), maxModel)
	Text(screen, x, y, w-x, suffix, tcell.StyleDefault.Dim(true))
}
