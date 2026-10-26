// Purpose: Apply BIOS-style row navigation, inline editing and y/n/Esc save decisions.
// Pattern: Event-loop input transitions.
// Usage: The model settings page routes normalized keys here before generic menu handling.
// Related: internal/tui/session/model_form.go, internal/tui/session/model_form_setup.go, internal/tui/input/keys.go.
package session

import (
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/input"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
)

func (a *app) modelSettingsKey(e *tcell.EventKey) {
	if a.modelForm == nil {
		a.initializeModelForm()
	}
	f := a.modelForm
	if f.Confirm {
		switch confirmationKey(e) {
		case confirmEdit:
			f.Confirm = false
		case confirmAccept:
			a.saveModelForm()
		case confirmReject:
			a.closeModelForm()
			a.page, a.status = "chat", ""
		}
		return
	}
	if e.Key() == tcell.KeyEnter && !a.worker.agent() && !a.worker.stopping() {
		if a.menuInput.String() == ":q" {
			a.closeModelForm()
			a.page = "chat"
			return
		}
		f.Confirm, f.Editing, f.Hints = true, false, nil
		a.menuInput.Set("")
		return
	}
	if a.worker.active() {
		if e.Key() == tcell.KeyEscape && !a.worker.agent() && a.taskCancel != nil {
			a.worker.stop()
			a.taskCancel()
			return
		}
		if !f.Catalog || !f.Editing || f.Row != modelsettings.Name {
			return
		}
	}
	if e.Key() == tcell.KeyEscape {
		if f.Details {
			f.Details = false
			return
		}
		if f.Editing {
			f.Editing = false
			f.Hints = nil
			return
		}
		a.closeModelForm()
		a.page = "chat"
		return
	}
	if f.Editing && f.Row == modelsettings.Name && len(f.Hints) > 0 {
		switch e.Key() {
		case tcell.KeyUp:
			f.Hint = max(0, f.Hint-1)
			return
		case tcell.KeyDown:
			f.Hint = min(len(f.Hints)-1, f.Hint+1)
			return
		case tcell.KeyTab:
			f.Name.Set(f.Hints[f.Hint])
			f.Invalidate(false)
			f.Editing, f.Hints = false, nil
			return
		}
	}
	switch e.Key() {
	case tcell.KeyPgUp, tcell.KeyPgDn:
		if f.Details {
			if e.Key() == tcell.KeyPgUp {
				f.DetailsOffset = max(0, f.DetailsOffset-8)
			} else {
				f.DetailsOffset += 8
			}
		}
	case tcell.KeyUp, tcell.KeyDown, tcell.KeyTab:
		delta := 1
		if e.Key() == tcell.KeyUp {
			delta = -1
		}
		wasModel := f.Row == modelsettings.Name && f.Editing
		f.Row, f.Editing, f.Hints = (f.Row+delta+modelsettings.RowCount)%modelsettings.RowCount, false, nil
		if wasModel && f.Prepared == nil && strings.TrimSpace(f.Name.String()) != "" {
			a.queryModelForm(strings.TrimSpace(f.Name.String()))
		}
	case tcell.KeyRight, tcell.KeyLeft:
		if f.Editing {
			input.EditKey(f.Editor(), e)
			return
		}
		direction := 1
		if e.Key() == tcell.KeyLeft {
			direction = -1
		}
		a.changeModelRow(direction)
	default:
		if f.Editing {
			if f.Edit(e) {
				a.status = ""
			}
			return
		}
		if strings.HasPrefix(e.Str(), "!") || strings.HasPrefix(e.Str(), ":") || a.menuInput.String() != "" {
			input.EditKey(&a.menuInput, e)
		}
	}
}

func (a *app) refreshModelHints() {
	if a.modelForm != nil && a.page == "model" {
		a.modelForm.RefreshHints()
	}
}

func (a *app) changeModelRow(direction int) {
	f := a.modelForm
	action, notice := f.Change(direction)
	if notice != "" {
		a.status = a.language.Text(notice)
		return
	}
	switch action {
	case modelsettings.ListModels:
		a.queryModelForm("")
	case modelsettings.InspectModel:
		a.queryModelForm(strings.TrimSpace(f.Name.String()))
	case modelsettings.DiagnoseConnection:
		selected := f.Selection()
		if selected.Connection.ID != a.selection.Connection.ID || selected.Model != a.selection.Model || f.ControlsChanged || f.KeyChanged {
			a.status = a.language.Text("Save these settings before running diagnostics")
			return
		}
		a.diagnoseModel()
	}
}
