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
	if f == nil || a.busy {
		return
	}
	c, err := f.connection()
	if err != nil {
		a.status = a.language.Text(err.Error())
		return
	}
	key := f.credential.String()
	a.opts.Vault.Remember(key)
	f.catalog = model == ""
	revision := f.revision
	if f.catalog {
		revision = f.connectionRevision
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
	expected := f.revision
	if f.catalog {
		expected = f.connectionRevision
	}
	f.catalog = false
	if n.FormRevision != expected {
		if n.Client != nil && n.Client != a.client {
			n.Client.Close()
		}
		f.saving, a.page = false, "model"
		a.status = a.language.Text("Settings changed; retry model lookup")
		return
	}
	if n.Err != nil {
		if n.Client != nil && n.Client != a.client {
			n.Client.Close()
		}
		if f.saving && errors.Is(n.Err, context.Canceled) {
			a.closeModelForm()
			a.page = "chat"
			return
		}
		f.saving = false
		a.page = "model"
		return
	}
	if n.Models != nil {
		f.models = n.Models
		f.refreshHints()
		if f.saving {
			a.saveModelForm()
		}
		return
	}
	if n.Selection == nil || n.Client == nil {
		return
	}
	if f.client != nil && f.client != a.client {
		f.client.Close()
	}
	f.client, f.prepared = n.Client, n.Selection
	if !f.controlsChanged {
		f.applyDefaults()
	}
	if f.saving {
		a.commitModelForm()
	}
}

func (a *app) saveModelForm() {
	f := a.modelForm
	if f == nil {
		return
	}
	if _, err := f.connection(); err != nil {
		f.confirm = false
		a.status = a.language.Text(err.Error())
		return
	}
	model := strings.TrimSpace(f.name.String())
	if model == "" {
		f.confirm = false
		a.status = a.language.Text("Enter a model ID before saving")
		return
	}
	if a.busy {
		if !a.agentActive {
			f.confirm, f.editing, f.saving = false, false, true
			a.page = "chat"
		}
		return
	}
	f.confirm, f.editing, f.saving = false, false, true
	next := f.selection()
	prepared := f.prepared != nil && f.prepared.Connection.ID == next.Connection.ID && f.prepared.Model == model
	active := a.client != nil && a.selection.Connection.ID == next.Connection.ID && a.selection.Model == model
	if prepared && (f.client != nil || active) {
		a.commitModelForm()
		return
	}
	a.queryModelForm(model)
	if a.busy {
		a.page = "chat"
	} else {
		f.saving = false
	}
}

func (a *app) commitModelForm() {
	f := a.modelForm
	next := f.selection()
	if err := selection.Validate(next); err != nil {
		f.saving, a.page = false, "model"
		a.fail(err)
		return
	}
	client := f.client
	if client == nil {
		client = a.client
	}
	f.client = nil // adoptConnection now owns this candidate's failure cleanup.
	adopted, err := a.adoptConnection(notice{Client: client, Selection: &next, credential: f.credential.String(), RememberModel: f.global})
	if err != nil && !adopted {
		f.saving, a.page = false, "model"
		a.fail(err)
		return
	}
	a.closeModelForm()
	a.page, a.status = "chat", ""
	if err != nil {
		a.fail(err)
	}
}
