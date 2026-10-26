// Purpose: Paint model settings and bind the visible editable value to input selection.
// Pattern: Screen adapter.
// Usage: Model page rendering composes the settings component with current runtime status.
// Related: internal/tui/settings/model/display.go, internal/tui/session/input_region.go.
package session

import (
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
	"github.com/keonho-kim/meca/internal/tui/view"
)

func (a *app) renderModel(w, h int) {
	if a.modelForm == nil {
		a.initializeModelForm()
	}
	f := a.modelForm
	display := f.Display(w, a.opts.Config.ContextLength, a.language, a.opts.Vault.Redact)
	display.Activity, display.Usage, display.Status, display.Command = a.activity(), a.displayTokens(), a.status, a.menuInput.String()
	fieldX := view.SettingsValueColumn(w)
	y := view.Settings(a.screen, display, w, h)
	if display.Field != nil && f.Row != modelsettings.Credential && y >= 0 {
		a.inputRegion = inputRegion{editor: f.Editor(), window: *display.Field, x: fieldX, y: y, width: w - fieldX, revision: f.Editor().Revision()}
	}
}
