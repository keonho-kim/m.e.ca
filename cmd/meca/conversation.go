// Purpose: Own one terminal session's database lease, recovery and tool resources until every worker has stopped.
// Pattern: Composition Root resource lifecycle.
// Usage: start calls runConversation and creates a new store only after it returns.
// Related: cmd/meca/main.go, internal/tui/session/app.go, internal/storage/sqlite/locks_unix.go.
package main

import (
	"context"
	"errors"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/storage/recovery"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
	tuisession "github.com/keonho-kim/meca/internal/tui/session"
)

func runConversation(ctx context.Context, screen tcell.Screen, configuration config.Config, home, state string, store *sqlite.Store) (action tuisession.Action, err error) {
	session := store.Session()
	defer func() { err = errors.Join(err, store.Close()) }()
	lease, leaseErr := store.LeaseSession()
	if leaseErr != nil {
		return tuisession.Exit, leaseErr
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	if err = recovery.Restore(ctx, store, session.ID); err != nil {
		return tuisession.Exit, err
	}
	vault := &credentials.Vault{State: state}
	executor, err := local.NewExecutor(session.ProjectRoot, home, state, vault.Redact)
	if err != nil {
		return tuisession.Exit, err
	}
	defer func() { err = errors.Join(err, executor.Close()) }()
	return tuisession.Run(ctx, screen, tuisession.Options{Home: home, State: state, Config: configuration, Store: store, Session: session, Tools: executor, Files: executor.Files, Vault: vault, LoadPrompts: func() (prompt.Templates, error) { return prompts.Load(state) }, LoadLAT: func() (prompt.Templates, error) { return prompts.LoadLAT(state) }, SaveConfig: func(c config.Config) error { return config.Save(state, c) }})
}
