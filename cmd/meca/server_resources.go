//go:build !minimum

// Purpose: Compose one immutable project runtime with shared process budgets.
// Pattern: Composition root.
// Usage: The workspace registry opens and closes these resources lazily.
// Related: cmd/meca/server.go, cmd/meca/server_session.go, cmd/meca/server_selection.go.
package main

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/diagnostics"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
	"github.com/keonho-kim/meca/internal/serverapi/detail"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/storage/serverrequests"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
	"github.com/keonho-kim/meca/internal/tools/local/filesystem"

	"github.com/keonho-kim/meca/internal/provider/diagnostic"
	"github.com/keonho-kim/meca/internal/tools/workspacegate"
)

type projectDependencies struct {
	home, state string
	sessions    *sqlite.Sessions
	vault       *credentials.Vault
	pages       *detail.Cache
	feed        *server.Feed
	capacity    *server.Capacity
	effects     *workspacegate.Gate
	attachments *server.AttachmentPool
	current     func() config.Config
	update      func(context.Context, func(*config.Config)) (config.Config, error)
}

func (d projectDependencies) open(ctx context.Context, mode startup.Mode, workspaceID string) (runtime server.WorkspaceRuntime, err error) {
	cfg, home, state := d.current(), d.home, d.state
	sessions, vault, pages := d.sessions, d.vault, d.pages
	feed, capacity, effects, attachments := d.feed, d.capacity, d.effects, d.attachments
	current, update := d.current, d.update
	var cleanup []func() error
	closeAll := func() error {
		var result error
		for i := len(cleanup) - 1; i >= 0; i-- {
			result = errors.Join(result, cleanup[i]())
		}
		return result
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, closeAll())
		}
	}()
	owner, err := sessions.LeaseServer(mode.Project)
	if err != nil {
		return
	}
	cleanup = append(cleanup, owner.Close)
	if err = sessions.CleanupDeleted(ctx, mode.Project); err != nil {
		return
	}
	ledger, err := serverrequests.Open(ctx, filepath.Join(state, "server-requests"), server.ProjectScope(mode.Project))
	if err != nil {
		return
	}
	cleanup = append(cleanup, ledger.Close)
	ctx = diagnostics.WithFields(diagnostics.WithRedactor(ctx, vault.Redact), "workspace_id", workspaceID)
	skillBoundary, err := filesystem.New(mode.Project, home, state)
	if err != nil {
		return
	}
	skillReader := skills.New(mode.Project, home, skills.Options{Redact: vault.Redact, Check: skillBoundary.SkillTarget, RedactRange: vault.RedactRange, Version: vault.RedactionVersion})
	files, err := local.NewExecutor(mode.Project, home, state, vault.Redact, local.Options{Skills: skillReader, RedactRange: vault.RedactRange, RedactionVersion: vault.RedactionVersion, NewChunkMask: vault.NewChunkMask})
	if err != nil {
		return
	}
	cleanup = append(cleanup, files.Close)
	application, err := server.New(ctx, server.Options{
		Attachments: attachments, WorkspaceID: workspaceID, Feed: feed, Capacity: capacity, Effects: effects, CurrentConfig: current,
		Delete: func(ctx context.Context, id string) error { return sessions.DeleteConversation(ctx, mode.Project, id) },
		Diagnose: func(ctx context.Context, selected connection.Selection, key string) (connection.Capabilities, error) {
			if err := vault.Remember(key); err != nil {
				return connection.Capabilities{}, err
			}
			prepared, err := setup.Prepare(ctx, selected.Connection, key, selected.Model, setup.Defaults{ContextLength: current().ContextLength}, selected, vault)
			if err != nil {
				return connection.Capabilities{}, err
			}
			defer prepared.Client.Close()
			settings := current().InferenceSettings()
			native := prepared.Selection.Details != nil && prepared.Selection.Details.Reasoning != nil && prepared.Selection.Details.Reasoning.Enabled()
			return diagnostic.Check(ctx, prepared.Client, selected.Model, prepared.Selection.ThinkingSupported, native, &settings)
		},
		Files: files.Files,
		Read: func(ctx context.Context, id string) (*server.Conversation, error) {
			store, err := sessions.OpenRead(ctx, mode.Project, id)
			if err != nil {
				return nil, err
			}
			selected, err := store.Selection(ctx, id)
			if err != nil {
				return nil, errors.Join(err, store.Close())
			}
			selected, err = serverSelection(selected, current().Model)
			if err != nil {
				return nil, errors.Join(err, store.Close())
			}
			team, err := store.TeamMode(ctx)
			if err != nil {
				return nil, errors.Join(err, store.Close())
			}
			return &server.Conversation{Runtime: agentrequest.Runtime{Session: store.Session(), Selection: selected}, History: store, TeamEnabled: team, Close: store.Close}, nil
		},
		Discover: func(ctx context.Context, selected connection.Selection, key string, defaults setup.Defaults) ([]connection.ModelInfo, *connection.Selection, error) {
			if err := vault.Remember(key); err != nil {
				return nil, nil, err
			}
			prepared, err := setup.Prepare(ctx, selected.Connection, key, selected.Model, defaults, selected, vault)
			if prepared.Client != nil {
				prepared.Client.Close()
			}
			return prepared.Models, prepared.Selection, err
		},
		Concurrency: mode.ServerOptions.Concurrency,
		Config:      cfg, Ledger: ledger, Skills: skillReader,
		Open: func(ctx context.Context, id string, create bool, defaults config.Config) (*server.Conversation, error) {
			c, err := openServerSession(ctx, mode, defaults, home, state, sessions, vault, skillReader, id, create)
			if err != nil {
				return nil, err
			}
			closeSession := c.Close
			c.Close = func() error { return errors.Join(pages.CloseSession(c.Runtime.Session.ID), closeSession()) }
			return c, nil
		},
		List: func(ctx context.Context, cursor string, limit int) ([]sessionstate.Conversation, string, error) {
			return sessions.ConversationPage(ctx, mode.Project, cursor, limit)
		},
		PrepareModel: func(ctx context.Context, selected connection.Selection, credential string, defaults setup.Defaults) (server.PreparedModel, error) {
			return prepareServerModel(ctx, selected, credential, defaults, vault)
		},
		UpdateConfig: update,
	})
	if err != nil {
		return
	}
	cleanup = append(cleanup, application.Close)
	if mode.Session != "" {
		if _, selectErr := application.Select(ctx, mode.Session, false); selectErr != nil {
			err = selectErr
			return
		}
	}
	runtime = server.WorkspaceRuntime{Service: application, Close: closeAll}
	return
}
