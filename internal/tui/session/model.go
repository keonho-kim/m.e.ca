// Purpose: Own cancellable connection workers and transfer setup results to the session.
// Pattern: Use Case with explicit worker lifecycle.
// Usage: Started by model menus; transfers completed clients through ordered session notices.
// Related: internal/provider/connection/connection.go, internal/provider/setup/setup.go, internal/tui/session/events.go.
package session

import (
	"context"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/setup"
)

func (a *app) connect(connection modelconn.Connection, key, model string, remember bool) {
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
	previous := a.selection
	defaults := setup.Defaults{ContextLength: a.opts.Config.ContextLength, Thinking: a.opts.Config.Thinking}
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
		prepared, err := setup.Prepare(ctx, connection, key, model, defaults, previous, a.opts.Vault)
		n := notice{SetupRevision: revision, Client: prepared.Client, Selection: prepared.Selection, Err: err, RememberModel: remember}
		if connection.CredentialRef == "" {
			n.credential = key
		}
		if model == "" && prepared.Selection != nil {
			n.Models = make([]string, 0, len(prepared.Models))
			for _, model := range prepared.Models {
				n.Models = append(n.Models, model.ID)
			}
		}
		finish(n)
	}()
}
