// Purpose: Apply BIOS-style row navigation, inline editing and y/n/Esc save decisions.
// Pattern: Event-loop input transitions.
// Usage: The model settings page routes normalized keys here before generic menu handling.
// Related: internal/tui/session/model_form.go, internal/tui/session/model_form_setup.go, internal/tui/input/keys.go.
package session

import (
	"slices"
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tui/input"
	"github.com/keonho-kim/meca/internal/tui/modelmenu"
)

func (a *app) modelSettingsKey(e *tcell.EventKey) {
	if a.modelForm == nil {
		a.initializeModelForm()
	}
	f := a.modelForm
	if f.confirm {
		switch confirmationKey(e) {
		case confirmEdit:
			f.confirm = false
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
		f.confirm, f.editing, f.hints = true, false, nil
		a.menuInput.Set("")
		return
	}
	if a.worker.active() {
		if e.Key() == tcell.KeyEscape && !a.worker.agent() && a.taskCancel != nil {
			a.worker.stop()
			a.taskCancel()
			return
		}
		if !f.catalog || !f.editing || f.row != modelName {
			return
		}
	}
	if e.Key() == tcell.KeyEscape {
		if f.details {
			f.details = false
			return
		}
		if f.editing {
			f.editing = false
			f.hints = nil
			return
		}
		a.closeModelForm()
		a.page = "chat"
		return
	}
	if f.editing && f.row == modelName && len(f.hints) > 0 {
		switch e.Key() {
		case tcell.KeyUp:
			f.hint = max(0, f.hint-1)
			return
		case tcell.KeyDown:
			f.hint = min(len(f.hints)-1, f.hint+1)
			return
		case tcell.KeyTab:
			f.name.Set(f.hints[f.hint])
			f.invalidate(false)
			f.editing, f.hints = false, nil
			return
		}
	}
	switch e.Key() {
	case tcell.KeyPgUp, tcell.KeyPgDn:
		if f.details {
			if e.Key() == tcell.KeyPgUp {
				f.detailsOffset = max(0, f.detailsOffset-8)
			} else {
				f.detailsOffset += 8
			}
		}
	case tcell.KeyUp, tcell.KeyDown, tcell.KeyTab:
		delta := 1
		if e.Key() == tcell.KeyUp {
			delta = -1
		}
		wasModel := f.row == modelName && f.editing
		f.row, f.editing, f.hints = (f.row+delta+modelRowCount)%modelRowCount, false, nil
		if wasModel && f.prepared == nil && strings.TrimSpace(f.name.String()) != "" {
			a.queryModelForm(strings.TrimSpace(f.name.String()))
		}
	case tcell.KeyRight, tcell.KeyLeft:
		if f.editing {
			input.EditKey(f.editor(), e)
			return
		}
		direction := 1
		if e.Key() == tcell.KeyLeft {
			direction = -1
		}
		a.changeModelRow(direction)
	default:
		if f.editing {
			a.editModelField(e)
			return
		}
		if strings.HasPrefix(e.Str(), "!") || strings.HasPrefix(e.Str(), ":") || a.menuInput.String() != "" {
			input.EditKey(&a.menuInput, e)
		}
	}
}

func (a *app) editModelField(e *tcell.EventKey) {
	f := a.modelForm
	editor := f.editor()
	if editor == nil {
		return
	}
	before := editor.Revision()
	if e.Key() == tcell.KeyCtrlU {
		editor.Set("")
	} else {
		input.EditKey(editor, e)
	}
	clearedKey := f.row == modelCredential && (e.Key() == tcell.KeyCtrlU || e.Key() == tcell.KeyBackspace || e.Key() == tcell.KeyDelete)
	if editor.Revision() != before || clearedKey {
		if f.row == modelCredential {
			f.keyChanged = true
		}
		f.invalidate(f.row != modelName)
		a.status = ""
	}
	f.refreshHints()
}

func (f *modelForm) refreshHints() {
	if f.editing && f.row == modelName {
		f.hints = modelmenu.Suggestions(f.models, f.name.String())
		f.hint = 0
	}
}

func (a *app) refreshModelHints() {
	if a.modelForm != nil && a.page == "model" {
		a.modelForm.refreshHints()
	}
}

func (a *app) changeModelRow(direction int) {
	f := a.modelForm
	options := f.options()
	switch f.row {
	case modelProvider:
		providers := connection.Providers()
		f.provider = (f.provider + direction + len(providers)) % len(providers)
		f.endpoint.Set(providers[f.provider].Endpoint)
		f.name.Set("")
		f.credential.Set("")
		f.keyChanged = true
		f.invalidate(true)
	case modelEndpoint, modelCredential, modelName:
		if direction < 0 {
			return
		}
		f.editing = true
		if f.row == modelName {
			f.refreshHints()
			if f.models == nil {
				a.queryModelForm("")
			}
		}
	case modelThinking:
		if f.prepared == nil && strings.TrimSpace(f.name.String()) != "" {
			a.queryModelForm(strings.TrimSpace(f.name.String()))
			return
		}
		values := []bool{}
		if options.Enabled() {
			values = append(values, true)
		}
		if options.Off {
			values = append(values, false)
		}
		if len(values) == 0 {
			a.status = a.language.Text("Thinking ON/OFF control is unavailable or unverified for this model")
			return
		}
		index := -1
		if f.thinking != nil {
			index = slices.Index(values, *f.thinking)
		}
		if index < 0 && direction < 0 {
			index = 0
		}
		value := values[(index+direction+len(values))%len(values)]
		f.thinking, f.controlsChanged = &value, true
	case modelEffort:
		if f.prepared == nil && strings.TrimSpace(f.name.String()) != "" {
			a.queryModelForm(strings.TrimSpace(f.name.String()))
			return
		}
		if len(options.Levels) == 0 {
			a.status = a.language.Text("Thinking effort is unavailable for this connection")
			return
		}
		index := slices.Index(options.Levels, f.effort)
		if index < 0 && direction < 0 {
			index = 0
		}
		f.effort = options.Levels[(index+direction+len(options.Levels))%len(options.Levels)]
		if f.thinking == nil {
			on := true
			f.thinking = &on
		}
		f.controlsChanged = true
	case modelDestination:
		f.global = !f.global
	case modelCatalog:
		a.queryModelForm("")
	case modelInformation:
		f.details = !f.details
	case modelDiagnostic:
		selected := f.selection()
		if selected.Connection.ID != a.selection.Connection.ID || selected.Model != a.selection.Model || f.controlsChanged || f.keyChanged {
			a.status = a.language.Text("Save these settings before running diagnostics")
			return
		}
		a.diagnoseModel()
	}
}
