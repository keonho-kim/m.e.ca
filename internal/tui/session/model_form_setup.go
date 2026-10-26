// Purpose: Prepare model-setting candidates without adopting them before explicit save confirmation.
// Pattern: Event-loop setup and commit transitions.
// Usage: Model form actions start metadata workers; worker results transfer client ownership here.
// Related: internal/tui/session/model.go, internal/tui/session/connection.go, internal/provider/setup/setup.go.
package session

import (
	"context"
	"errors"
	"strings"

	"github.com/keonho-kim/meca/internal/provider/selection"
	"github.com/keonho-kim/meca/internal/provider/setup"
)

func (a *app) queryModelForm(model string) {
	f := a.modelForm
	if f == nil || a.worker.active() {
		return
	}
	c, err := f.Connection()
	if err != nil {
		a.status = a.language.Text(err.Error())
		return
	}
	key := f.Credential.String()
	a.opts.Vault.Remember(key)
	f.Catalog = model == ""
	revision := f.Revision
	if f.Catalog {
		revision = f.ConnectionRevision
	}
	a.startModelSetup(modelSetupRequest{connection: c, key: key, model: model, defaults: setup.Defaults{ContextLength: a.opts.Config.ContextLength}, form: true, formRevision: revision})
}

func (a *app) finishModelFormSetup(n notice) {
	f := a.modelForm
	if f == nil {
		if n.Client != nil && n.Client != a.client {
			n.Client.Close()
		}
		return
	}
	expected := f.Revision
	if f.Catalog {
		expected = f.ConnectionRevision
	}
	f.Catalog = false
	if n.FormRevision != expected {
		if n.Client != nil && n.Client != a.client {
			n.Client.Close()
		}
		f.Saving, a.page = false, "model"
		a.status = a.language.Text("Settings changed; retry model lookup")
		return
	}
	if n.Err != nil {
		if n.Client != nil && n.Client != a.client {
			n.Client.Close()
		}
		if f.Saving && errors.Is(n.Err, context.Canceled) {
			a.closeModelForm()
			a.page = "chat"
			return
		}
		f.Saving = false
		a.page = "model"
		return
	}
	if n.Models != nil {
		f.Models = n.Models
		f.RefreshHints()
		if f.Saving {
			a.saveModelForm()
		}
		return
	}
	if n.Selection == nil || n.Client == nil {
		return
	}
	if f.Client != nil && f.Client != a.client {
		f.Client.Close()
	}
	f.Client, f.Prepared = n.Client, n.Selection
	if !f.ControlsChanged {
		f.ApplyDefaults()
	}
	if f.Saving {
		a.commitModelForm()
	}
}

func (a *app) saveModelForm() {
	f := a.modelForm
	if f == nil {
		return
	}
	if _, err := f.Connection(); err != nil {
		f.Confirm = false
		a.status = a.language.Text(err.Error())
		return
	}
	model := strings.TrimSpace(f.Name.String())
	if model == "" {
		f.Confirm = false
		a.status = a.language.Text("Enter a model ID before saving")
		return
	}
	if a.worker.active() {
		if !a.worker.agent() {
			f.Confirm, f.Editing, f.Saving = false, false, true
			a.page = "chat"
		}
		return
	}
	f.Confirm, f.Editing, f.Saving = false, false, true
	next := f.Selection()
	prepared := f.Prepared != nil && f.Prepared.Connection.ID == next.Connection.ID && f.Prepared.Model == model
	active := a.client != nil && a.selection.Connection.ID == next.Connection.ID && a.selection.Model == model
	if prepared && (f.Client != nil || active) {
		a.commitModelForm()
		return
	}
	a.queryModelForm(model)
	if a.worker.active() {
		a.page = "chat"
	} else {
		f.Saving = false
	}
}

func (a *app) commitModelForm() {
	f := a.modelForm
	next := f.Selection()
	if err := selection.Validate(next); err != nil {
		f.Saving, a.page = false, "model"
		a.fail(err)
		return
	}
	client := f.Client
	if client == nil {
		client = a.client
	}
	f.Client = nil // adoptConnection now owns this candidate's failure cleanup.
	adopted, err := a.adoptConnection(notice{Client: client, Selection: &next, credential: f.Credential.String(), RememberModel: f.Global})
	if err != nil && !adopted {
		f.Saving, a.page = false, "model"
		a.fail(err)
		return
	}
	a.closeModelForm()
	a.page, a.status = "chat", ""
	if err != nil {
		a.fail(err)
	}
}
