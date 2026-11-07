// Purpose: Persist prepared model changes off the terminal loop and return durable adoption facts.
// Pattern: Joined setup worker with an explicit commit result.
// Usage: Connection preparation and confirmed settings forms commit through the same worker boundary.
// Related: internal/commands/modelchange/commit.go, internal/tui/session/connection.go.
package session

import (
	"context"
	"slices"

	"github.com/keonho-kim/meca/internal/commands/modelchange"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func (a *app) startModelCommit(n notice) {
	if a.work.state.active() {
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.work.cancel, a.work.state = cancel, workerSetup
	a.connection.setupRevision++
	n.SetupRevision = a.connection.setupRevision
	a.startElapsed()
	a.page = "chat"
	opts := a.opts
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		n = commitPreparedModel(ctx, opts, n)
		n.Done = true
		if a.post(a.ctx, n) != nil && n.Client != nil {
			n.Client.Close()
		}
	}()
}

func commitPreparedModel(ctx context.Context, opts Options, n notice) notice {
	if n.Err != nil || n.Selection == nil || n.Selection.Model == "" || n.Client == nil {
		return n
	}
	next := *n.Selection
	if next.Details != nil {
		value := *next.Details
		value.Notice = opts.Vault.Redact(value.Notice)
		next.Details = &value
	}
	if next.Identity != nil {
		value := *next.Identity
		value.Facts = slices.Clone(value.Facts)
		value.Redact(opts.Vault.Redact)
		next.Identity = &value
	}
	var remember func(connection.Selection) error
	if n.RememberModel {
		remember = func(selected connection.Selection) error {
			updated, err := opts.UpdateConfig(ctx, func(latest *config.Config) {
				latest.Model = config.RememberModel(selected)
				latest.Thinking = selected.Thinking
			})
			if err == nil {
				n.Config = &updated
			}
			return err
		}
	}
	result, err := modelchange.Commit(ctx, opts.Store, opts.Vault, opts.Session.ID, next, n.credential, remember)
	n.Commit, n.Err, n.credential = &result, err, ""
	return n
}
