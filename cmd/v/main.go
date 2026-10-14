// Purpose: Compose startup, private state, project adapters and terminal lifecycle.
// Pattern: Composition Root.
// Usage: go run ./cmd/v [path | resume]; go build -o bin/v ./cmd/v.
// Related: internal/commands/startup/parse.go, internal/config/bootstrap/bootstrap.go, internal/tui/session/app.go, cmd/v/sandbox_linux.go.
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"github.com/keonho-kim/v-code/internal/commands/startup"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/config/bootstrap"
	"github.com/keonho-kim/v-code/internal/config/credentials"
	"github.com/keonho-kim/v-code/internal/config/prompts"
	"github.com/keonho-kim/v-code/internal/storage/recovery"
	sessionstate "github.com/keonho-kim/v-code/internal/storage/session"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"github.com/keonho-kim/v-code/internal/tools/local"
	tuisession "github.com/keonho-kim/v-code/internal/tui/session"
	tuistartup "github.com/keonho-kim/v-code/internal/tui/startup"
	"golang.org/x/term"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
)

func main() {
	if handled, err := runSandboxChild(); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, "v: sandbox:", err)
		}
		os.Exit(125)
	}
	if err := start(); err != nil {
		fmt.Fprintln(os.Stderr, "v:", err)
		os.Exit(1)
	}
}
func start() (err error) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return fmt.Errorf("V currently supports Linux and macOS")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	mode, err := startup.Parse(os.Args[1:], cwd)
	if err != nil {
		return err
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("an interactive TTY is required")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return err
	}
	configuration, err := bootstrap.Initialize(home)
	if err != nil {
		return fmt.Errorf("bootstrap configuration: %w", err)
	}
	state := filepath.Join(home, ".v")
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		return fmt.Errorf("open session directory: %w", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err = screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()
	var store *sqlite.Store
	var session sessionstate.Session
	if mode.Resume {
		var selected bool
		session, selected, err = tuistartup.Resume(ctx, screen, sessions)
		if err != nil || !selected {
			return err
		}
		store, err = sessions.Open(ctx, session.ID)
		if err != nil {
			return err
		}
		if store.Session() != session {
			return errors.Join(fmt.Errorf("selected session metadata changed; select it again"), store.Close())
		}
	} else {
		store, err = sessions.Restore(ctx, mode.Project, mode.Display, configuration.Session.RestoreLast)
		if err != nil {
			return err
		}
		session = store.Session()
	}
	defer func() { err = errors.Join(err, store.Close()) }()
	lease, leaseErr := store.LeaseSession()
	if leaseErr != nil {
		return leaseErr
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	if err = recovery.Restore(ctx, store, session.ID); err != nil {
		return err
	}
	vault := &credentials.Vault{}
	executor, err := local.NewExecutor(session.ProjectRoot, home, state, vault.Redact)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, executor.Close()) }()
	return tuisession.Run(ctx, screen, tuisession.Options{Home: home, State: state, Config: configuration, Store: store, Session: session, Tools: executor, Vault: vault, LoadPrompts: func() (prompt.Templates, error) { return prompts.Load(state) }, SaveConfig: func(c config.Config) error { return config.Save(state, c) }})
}
