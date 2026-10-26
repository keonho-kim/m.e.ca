// Purpose: Apply model setting value changes and field edits before any connection I/O.
// Pattern: Settings state transitions with explicit I/O intents.
// Usage: The session dispatches returned lookup/diagnostic actions using its owned worker.
// Related: internal/tui/settings/model/form.go, internal/tui/session/model_form_keys.go.
package model

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tui/input"
	"slices"
	"strings"
)

type ControlAction uint8

const (
	NoAction ControlAction = iota
	ListModels
	InspectModel
	DiagnoseConnection
)

func (f *Form) Edit(e *tcell.EventKey) bool {
	editor := f.Editor()
	if editor == nil {
		return false
	}
	before := editor.Revision()
	if e.Key() == tcell.KeyCtrlU {
		editor.Set("")
	} else {
		input.EditKey(editor, e)
	}
	clearedKey := f.Row == Credential && (e.Key() == tcell.KeyCtrlU || e.Key() == tcell.KeyBackspace || e.Key() == tcell.KeyDelete)
	changed := editor.Revision() != before || clearedKey
	if changed {
		if f.Row == Credential {
			f.KeyChanged = true
		}
		f.Invalidate(f.Row != Name)
	}
	f.RefreshHints()
	return changed
}

func (f *Form) Change(direction int) (ControlAction, string) {
	options := f.Options()
	switch f.Row {
	case Provider:
		providers := connection.Providers()
		f.Provider = (f.Provider + direction + len(providers)) % len(providers)
		f.Endpoint.Set(providers[f.Provider].Endpoint)
		f.Name.Set("")
		f.Credential.Set("")
		f.KeyChanged = true
		f.Invalidate(true)
	case Endpoint, Credential, Name:
		if direction < 0 {
			return NoAction, ""
		}
		f.Editing = true
		if f.Row == Name {
			f.RefreshHints()
			return ListModels, ""
		}
	case Thinking:
		if f.Prepared == nil && strings.TrimSpace(f.Name.String()) != "" {
			return InspectModel, ""
		}
		values := []bool{}
		if options.Enabled() {
			values = append(values, true)
		}
		if options.Off {
			values = append(values, false)
		}
		if len(values) == 0 {
			return NoAction, "Thinking ON/OFF control is unavailable or unverified for this model"
		}
		index := -1
		if f.Thinking != nil {
			index = slices.Index(values, *f.Thinking)
		}
		if index < 0 && direction < 0 {
			index = 0
		}
		value := values[(index+direction+len(values))%len(values)]
		f.Thinking, f.ControlsChanged = &value, true
	case Effort:
		if f.Prepared == nil && strings.TrimSpace(f.Name.String()) != "" {
			return InspectModel, ""
		}
		if len(options.Levels) == 0 {
			return NoAction, "Thinking effort is unavailable for this connection"
		}
		index := slices.Index(options.Levels, f.Effort)
		if index < 0 && direction < 0 {
			index = 0
		}
		f.Effort = options.Levels[(index+direction+len(options.Levels))%len(options.Levels)]
		if f.Thinking == nil {
			on := true
			f.Thinking = &on
		}
		f.ControlsChanged = true
	case Destination:
		f.Global = !f.Global
	case Information:
		f.Details = !f.Details
	case Diagnostic:
		return DiagnoseConnection, ""
	}
	return NoAction, ""
}
