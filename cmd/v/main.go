// Purpose: Compose startup, private state, project adapters and terminal lifecycle.
// Pattern: Composition Root.
// Usage: go run ./cmd/v [path | resume]; go build -o bin/v ./cmd/v.
// Related: internal/commands/startup/parse.go, internal/config/bootstrap/bootstrap.go, internal/tui/session/app.go.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/keonho-kim/v-code/internal/commands/startup"
	"github.com/keonho-kim/v-code/internal/config/bootstrap"
	"github.com/keonho-kim/v-code/internal/config/credentials"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"github.com/keonho-kim/v-code/internal/tools/local"
	tuisession "github.com/keonho-kim/v-code/internal/tui/session"
	tuistartup "github.com/keonho-kim/v-code/internal/tui/startup"
	"golang.org/x/term"
)

func main() {
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
	store, err := sqlite.Open(filepath.Join(state, "v.db"))
	if err != nil {
		return fmt.Errorf("bootstrap database: %w", err)
	}
	defer func() { err = errors.Join(err, store.Close()) }()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err = screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()
	var session sqlite.Session
	if mode.Resume {
		var selected bool
		session, selected, err = tuistartup.Resume(ctx, screen, store)
		if err != nil || !selected {
			return err
		}
	} else {
		session, err = store.RestoreSession(ctx, mode.Project, mode.Display, configuration.Session.RestoreLast)
		if err != nil {
			return err
		}
	}
	lease, leaseErr := store.LeaseSession(session.ID)
	if leaseErr != nil {
		return leaseErr
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	if err = store.Recover(ctx, session.ID); err != nil {
		return err
	}
	vault := &credentials.Vault{}
	executor, err := local.NewExecutor(session.ProjectRoot, home, state, vault.Redact)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, executor.Close()) }()
	return tuisession.Run(ctx, screen, tuisession.Options{Home: home, State: state, Config: configuration, Store: store, Session: session, Tools: executor, Vault: vault})
}
