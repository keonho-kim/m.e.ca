// Purpose: Compose maintenance, headless and terminal startup with owned session transitions.
// Pattern: Composition Root.
// Usage: v --help lists maintenance, interactive and headless invocations.
// Related: cmd/v/install.go, cmd/v/headless.go, internal/commands/startup/parse.go, internal/tui/session/app.go, cmd/v/conversation.go.
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/commands/startup"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/config/bootstrap"
	"github.com/keonho-kim/v-code/internal/config/statefs"
	sessionstate "github.com/keonho-kim/v-code/internal/storage/session"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	tuisession "github.com/keonho-kim/v-code/internal/tui/session"
	"golang.org/x/term"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
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
	if mode.Action == startup.Help {
		fmt.Fprintln(os.Stdout, "Usage: v [path] [--print|--mode json] [--session ID] [prompt]\n       v --session ID --export\n       v --diagnose\n       v resume | install | uninstall")
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
		return maintenance(ctx, mode.Action, home)
	}
	if err := startup.CheckAgentUser(os.Geteuid()); err != nil {
		return err
	}
	root, err := statefs.Open(filepath.Join(home, ".v"))
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
	state := filepath.Join(home, ".v")
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
	store, selected, err := selectConversation(ctx, screen, sessions, mode, cwd)
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
