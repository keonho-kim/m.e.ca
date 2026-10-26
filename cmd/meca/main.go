// Purpose: Compose maintenance, headless and terminal startup with owned session transitions.
// Pattern: Composition Root.
// Usage: meca --help lists maintenance, interactive and headless invocations.
// Related: cmd/meca/install.go, cmd/meca/headless.go, internal/commands/startup/parse.go, internal/tui/session/app.go, cmd/meca/conversation.go.
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/statefs"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	tuisession "github.com/keonho-kim/meca/internal/tui/session"
	"golang.org/x/term"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
)

func main() {
	if err := start(); err != nil {
		fmt.Fprintln(os.Stderr, "meca:", err)
		os.Exit(1)
	}
}
func start() (err error) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return fmt.Errorf("MECA currently supports Linux and macOS")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	mode, err := startup.Parse(os.Args[1:], cwd)
	if err != nil {
		return err
	}
	if mode.Action == startup.Help {
		fmt.Fprintln(os.Stdout, "Usage: meca [path] [--print|--mode json] [--session ID] [prompt]\n       meca --session ID --export\n       meca --diagnose\n       meca install [--no-shell]\n       meca resume | uninstall")
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	if mode.Action != "" {
		return maintenance(ctx, mode, home)
	}
	if err := startup.CheckAgentUser(os.Geteuid()); err != nil {
		return err
	}
	root, err := statefs.Open(filepath.Join(home, statefs.DirectoryName))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	lease, err := statefs.Lease(root, false)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	configuration, err := bootstrap.Initialize(home)
	if err != nil {
		return fmt.Errorf("bootstrap configuration: %w", err)
	}
	state := filepath.Join(home, statefs.DirectoryName)
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		return fmt.Errorf("open session directory: %w", err)
	}
	if mode.Print {
		return runHeadless(ctx, mode, configuration, home, state, sessions)
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("use --print or --mode json without an interactive TTY")
	}
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err = screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()
	store, selected, err := selectConversation(ctx, screen, sessions, mode, cwd, configuration.Language)
	if err != nil || !selected {
		return err
	}
	var session sessionstate.Session
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
