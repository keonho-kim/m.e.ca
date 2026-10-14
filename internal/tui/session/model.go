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

func (a *app) connect(connection modelconn.Connection, key, model string) {
	if a.busy {
		return
	}
	a.busy = true
	a.status = "Connecting"
	ctx, cancel := context.WithCancel(a.ctx)
	a.taskCancel = cancel
	previous := a.selection
	binding := a.opts.Config.ModelPreference(connection, model)
	inferenceSettings := a.opts.Config.Inference
	defaults := setup.Defaults{Inference: &inferenceSettings, Thinking: a.opts.Config.Thinking, Profile: binding.Profile, ReasoningEffort: binding.ReasoningEffort}
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
		n := notice{Client: prepared.Client, Selection: prepared.Selection, Err: err}
		if connection.CredentialRef == "" {
			n.credential = key
		}
		if model == "" && prepared.Selection != nil {
			for _, model := range prepared.Models {
				n.Models = append(n.Models, model.ID)
			}
			n.Models = append(n.Models, "Enter Model ID manually")
		}
		finish(n)
	}()
}
