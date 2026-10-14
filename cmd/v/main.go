// Purpose: Compose terminal startup and explicit transitions between independently owned sessions.
// Pattern: Composition Root.
// Usage: go run ./cmd/v [path | resume]; go build -o bin/v ./cmd/v.
// Related: internal/commands/startup/parse.go, internal/config/bootstrap/bootstrap.go, internal/tui/session/app.go, cmd/v/sandbox_linux.go, cmd/v/conversation.go.
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/commands/startup"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/config/bootstrap"
	sessionstate "github.com/keonho-kim/v-code/internal/storage/session"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
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
		store, err = sessions.Open(ctx, session.ProjectRoot, session.ID)
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
	for {
		session = store.Session()
		action, runErr := runConversation(ctx, screen, configuration, home, state, store)
		if runErr != nil {
			return runErr
		}
		if action != tuisession.NewSession || ctx.Err() != nil {
			return nil
		}
		configuration, err = config.Load(state)
		if err != nil {
			return err
		}
		store, err = sessions.Create(ctx, session.ProjectRoot, session.DisplayRoot)
		if err != nil {
			return err
		}
	}
}
