//go:build !tui

// Purpose: Compose multi-workspace serving with shared process budgets and optional UI.
// Pattern: Composition root.
// Usage: main runs meca --server [--ui] with the selected protocol.
// Related: cmd/meca/server_resources.go, internal/serverapi/network.go, internal/webui/assets.go.
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/desktop/folderpicker"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi"
	"github.com/keonho-kim/meca/internal/serverapi/detail"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
	"github.com/keonho-kim/meca/internal/tools/workspacegate"
	"github.com/keonho-kim/meca/internal/webui"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

func runServer(ctx context.Context, mode startup.Mode, cfg config.Config, home, state string, sessions *sqlite.Sessions) (err error) {
	canonical, canonicalErr := filesystem.Canonical(mode.Project)
	if canonicalErr != nil {
		return canonicalErr
	}
	mode.Project = canonical
	o := mode.ServerOptions
	if o.TokenFile == "" {
		o.TokenFile = filepath.Join(state, "server.token")
	}
	tokenLease, leaseErr := serverapi.LeaseTokenFile(o.TokenFile)
	if leaseErr != nil {
		return leaseErr
	}
	defer func() { err = errors.Join(err, tokenLease.Close()) }()
	var ui http.Handler
	if mode.ServerOptions.UI {
		ui, err = webui.New()
		if err != nil {
			return err
		}
	}
	vault := &credentials.Vault{State: state}
	ctx = diagnostics.WithRedactor(ctx, vault.Redact)
	pages := detail.NewCache()
	defer func() { err = errors.Join(err, pages.Close()) }()
	var current atomic.Value
	current.Store(cfg)
	snapshot := func() config.Config { return current.Load().(config.Config) }
	var configMu sync.Mutex
	update := func(ctx context.Context, apply func(*config.Config)) (config.Config, error) {
		configMu.Lock()
		defer configMu.Unlock()
		next, err := config.Update(ctx, state, apply)
		if err == nil {
			current.Store(next)
		}
		return next, err
	}
	attachments := server.NewAttachmentPool()
	boundary, err := filesystem.New(mode.Project, home, state)
	if err != nil {
		return err
	}
	check := func(path string) (string, error) {
		if path == "" {
			path = home
		}
		if path == "~" {
			path = home
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, path[2:])
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(mode.Project, path)
		}
		return boundary.ReadTarget(path)
	}
	feed, capacity, effects := server.NewFeed(), server.NewCapacity(mode.ServerOptions.Concurrency), &workspacegate.Gate{}
	projects := projectDependencies{home: home, state: state, sessions: sessions, vault: vault, pages: pages, feed: feed, capacity: capacity, effects: effects, attachments: attachments, current: snapshot, update: update}
	workspaces, err := server.NewWorkspaces(ctx, server.WorkspaceOptions{
		Default: mode.Project, Paths: cfg.Workspaces, Check: check,
		Save: func(ctx context.Context, path string) error {
			_, err := update(ctx, func(value *config.Config) {
				if !slices.Contains(value.Workspaces, path) {
					value.Workspaces = append(value.Workspaces, path)
				}
			})

			return err
		},
		Open: func(ctx context.Context, value server.Workspace) (server.WorkspaceRuntime, error) {
			scoped := mode
			scoped.Project = value.Path
			scoped.Display = value.Path
			if value.Path != mode.Project {
				scoped.Session = ""
			}
			return projects.open(ctx, scoped, value.ID)
		},
	})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, workspaces.Close()) }()
	application, release, err := workspaces.Acquire(ctx, "")
	if err != nil {
		return err
	}
	defer release()
	token := serverapi.NewToken()
	if err := vault.Remember(token); err != nil {
		return err
	}
	api := &serverapi.API{PickFolder: folderpicker.Pick, ScopeID: application.State().Workspace, Service: application, Workspaces: workspaces, Remember: vault.Remember, Redact: vault.Redact, RedactRange: vault.RedactRange, NewMask: vault.NewChunkMask, RedactPreview: func(text string) string { return vault.NewStreamMask()(text, true) }, RedactionVersion: vault.RedactionVersion, Details: pages}
	serveCtx, cancelServer := context.WithCancel(ctx)
	defer cancelServer()
	var startupErr error
	err = serverapi.ListenAndServe(serveCtx, api, token, serverapi.Options{Protocol: o.Protocol, Listen: o.Listen, Certificate: o.Certificate, Key: o.Key, Hosts: o.Hosts, Origins: o.Origins, UI: ui}, func(address string) {
		if publishErr := serverapi.PublishToken(o.TokenFile, token); publishErr != nil {
			startupErr = publishErr
			cancelServer()
			return
		}
		// Startup stdout is the explicit credential handoff to the user.
		if _, outputErr := fmt.Fprintln(os.Stdout, diag.Text(diag.ServerTokenReady, token)); outputErr != nil {
			startupErr = fault.New(diag.DeliveryFailed, diag.ServerStartupOutputFailed, outputErr)
			cancelServer()
			return
		}
		diagnostics.Log(ctx, diag.ServerListening, "component", "server", "protocol", o.Protocol, "listen", address)
		if o.UI {
			announceUI(ctx, address)
		}
	})
	return errors.Join(err, startupErr)
}
