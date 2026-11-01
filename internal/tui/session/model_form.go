// Purpose: Coordinate opening and closing the model settings component on the event loop.
// Pattern: Application lifecycle transitions.
// Usage: !model and connection completion retain or release one settings draft.
// Related: internal/tui/settings/model/form.go, internal/tui/session/model_form_setup.go.
package session

import (
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
)

func (a *app) openModel() {
	a.closeModelForm()
	a.stopNotice, a.status, a.page = "", "", "model"
	a.menuInput.Set("")
	a.initializeModelForm()
}

func (a *app) initializeModelForm() {
	a.connection.modelForm = modelsettings.New(a.connection.selection)
}

func (a *app) closeModelForm() {
	if f := a.connection.modelForm; f != nil {
		if a.work.state.active() && !a.work.state.agent() && a.work.cancel != nil {
			a.work.state.stop()
			a.work.cancel()
		}
		f.Close(a.connection.client)
	}
	a.connection.modelForm = nil
	a.menuInput.Set("")
}
