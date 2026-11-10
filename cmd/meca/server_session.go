//go:build !tui

// Purpose: Acquire exclusive server conversation resources using existing recovery and request factories.
// Pattern: Composition root resource factory.
// Usage: The server service adopts the returned conversation until close or replacement.
// Related: cmd/meca/server_selection.go, cmd/meca/conversation.go, cmd/meca/team.go.
package main

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/agent/session/recovery"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
)

func openServerSession(ctx context.Context, mode startup.Mode, cfg config.Config, home, state string, sessions *sqlite.Sessions, vault *credentials.Vault, skillReader *skills.Reader, id string, create bool) (value *server.Conversation, resultErr error) {
	var store *sqlite.Store
	var err error
	if create {
		store, err = sessions.Create(ctx, mode.Project, mode.Display)
	} else {
		store, err = sessions.Open(ctx, mode.Project, id)
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		if value == nil {
			resultErr = errors.Join(resultErr, store.Close())
		}
	}()
	lease, err := store.LeaseSession()
	if err != nil {
		return nil, err
	}
	defer func() {
		if value == nil {
			resultErr = errors.Join(resultErr, lease.Close())
		}
	}()
	if err := recovery.Restore(ctx, store, store.Session().ID); err != nil {
		return nil, err
	}
	selected, err := store.Selection(ctx, store.Session().ID)
	if err != nil {
		return nil, err
	}
	selected, err = serverSelection(selected, cfg.Model)
	if err != nil {
		return nil, err
	}
	executor, err := local.NewExecutor(mode.Project, home, state, vault.Redact, local.Options{Skills: skillReader, RedactRange: vault.RedactRange, RedactionVersion: vault.RedactionVersion, NewChunkMask: vault.NewChunkMask})
	if err != nil {
		return nil, err
	}
	defer func() {
		if value == nil {
			resultErr = errors.Join(resultErr, executor.Close())
		}
	}()
	team, err := store.TeamMode(ctx)
	if err != nil {
		return nil, err
	}
	c := &server.Conversation{Runtime: agentrequest.Runtime{Config: cfg, Selection: selected, Session: store.Session(), Tools: executor, Skills: executor.Skills(), Store: store, Vault: vault, LoadPrompts: func() (prompt.Templates, error) { return prompts.Load(state) }}, History: store, Models: store, Team: teamService(store, home, state, vault), TeamEnabled: team,
		Close: func() error { return errors.Join(executor.Close(), store.Close(), lease.Close()) },
	}
	// Opening history does not require a reachable provider or a working
	// keyring. Start prepares the saved selection before durable admission.
	return c, nil
}
