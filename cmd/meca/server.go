// Purpose: Compose server application resources and the selected network transport.
// Pattern: Composition root.
// Usage: main calls runServer for meca --server --protocol sse|socket|grpc.
// Related: cmd/meca/server_session.go, internal/commands/server/service.go, internal/serverapi/network.go.
package main

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
	"github.com/keonho-kim/meca/internal/serverapi"
	"github.com/keonho-kim/meca/internal/serverapi/detail"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/storage/serverrequests"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
)

func runServer(ctx context.Context, mode startup.Mode, cfg config.Config, home, state string, sessions *sqlite.Sessions) (err error) {
	owner, err := sessions.LeaseServer(mode.Project)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, owner.Close()) }()
	ledger, err := serverrequests.Open(ctx, filepath.Join(state, "server-requests"), server.ProjectScope(mode.Project))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, ledger.Close()) }()
	vault := &credentials.Vault{State: state}
	ctx = diagnostics.WithRedactor(ctx, vault.Redact)
	skillBoundary, err := filesystem.New(mode.Project, home, state)
	if err != nil {
		return err
	}
	skillReader := skills.New(mode.Project, home, skills.Options{Redact: vault.Redact, Check: skillBoundary.SkillTarget, RedactRange: vault.RedactRange, Version: vault.RedactionVersion})
	pages := detail.NewCache()
	defer func() { err = errors.Join(err, pages.Close()) }()
	application, err := server.New(ctx, server.Options{
		Config: cfg, Ledger: ledger, Responses: ledger, Project: mode.Project, Skills: skillReader,
		ReadOutput: func(ctx context.Context, id string, after, through int64, maxBytes int, reserve func(int) error) (string, error) {
			return sessions.ResponseOutput(ctx, mode.Project, id, after, through, maxBytes, reserve)
		},
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
		UpdateConfig: func(ctx context.Context, apply func(*config.Config)) (config.Config, error) {
			return config.Update(ctx, state, apply)
		},
	})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, application.Close()) }()
	if mode.Session != "" {
		if _, err := application.Select(ctx, mode.Session, false); err != nil {
			return err
		}
	}
	o := mode.ServerOptions
	if o.TokenFile == "" {
		o.TokenFile = filepath.Join(state, "server.token")
	}
	// Remember the server credential in the same masking vault as model keys.
	token, err := serverapi.LoadToken(o.TokenFile)
	if err != nil {
		return err
	}
	if err := vault.Remember(token); err != nil {
		return err
	}
	return serverapi.ListenAndServe(ctx, &serverapi.API{Service: application, Redact: vault.Redact, NewMask: vault.NewChunkMask, RedactPreview: func(text string) string { return vault.NewStreamMask()(text, true) }, RedactionVersion: vault.RedactionVersion, Details: pages}, token, serverapi.Options{Protocol: o.Protocol, Listen: o.Listen, Certificate: o.Certificate, Key: o.Key, Hosts: o.Hosts, Origins: o.Origins, OpenAI: o.OpenAI}, func(address string) {
		diagnostics.Log(ctx, diagcatalog.ServerListening, "component", "server", "protocol", o.Protocol, "listen", address)
	})
}
