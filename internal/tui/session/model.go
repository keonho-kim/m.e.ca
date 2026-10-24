// Purpose: Own cancellable connection workers and transfer setup results to the session.
// Pattern: Use Case with explicit worker lifecycle.
// Usage: Started by model menus; transfers completed clients through ordered session notices.
// Related: internal/provider/connection/connection.go, internal/provider/setup/setup.go, internal/tui/session/events.go.
package session

import (
	"context"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
)

func (a *app) connect(connection modelconn.Connection, key, model string, remember bool) {
	a.startModelSetup(modelSetupRequest{connection: connection, key: key, model: model, remember: remember, defaults: setup.Defaults{ContextLength: a.opts.Config.ContextLength, Thinking: a.opts.Config.Thinking}, previous: a.selection})
}

type modelSetupRequest struct {
	formRevision   int
	connection     modelconn.Connection
	key, model     string
	remember, form bool
	defaults       setup.Defaults
	previous       modelconn.Selection
}

func (a *app) startModelSetup(request modelSetupRequest) {
	if a.busy {
		return
	}
	a.busy = true
	a.startElapsed()
	a.status = a.language.Text("Connecting")
	ctx, cancel := context.WithCancel(a.ctx)
	a.taskCancel = cancel
	a.setupRevision++
	revision := a.setupRevision
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		finish := func(n notice) {
			n.Done = true
			if err := a.post(a.ctx, n); err != nil && n.Client != nil {
				n.Client.Close()
			}
		}
		prepared, err := setup.Prepare(ctx, request.connection, request.key, request.model, request.defaults, request.previous, a.opts.Vault)
		n := notice{SetupRevision: revision, Client: prepared.Client, Selection: prepared.Selection, Err: err, RememberModel: request.remember, ModelForm: request.form, FormRevision: request.formRevision}
		if request.connection.CredentialRef == "" {
			n.credential = request.key
		}
		if request.model == "" && prepared.Selection != nil {
			n.Models = make([]string, 0, len(prepared.Models))
			for _, model := range prepared.Models {
				n.Models = append(n.Models, model.ID)
			}
		}
		finish(n)
	}()
}
