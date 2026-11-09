// Purpose: Compose maintenance, headless, server and terminal startup with owned session transitions.
// Pattern: Composition Root.
// Usage: meca --help lists maintenance, interactive, headless and server invocations.
// Related: cmd/meca/install.go, cmd/meca/headless.go, internal/commands/startup/parse.go, internal/tui/session/app.go, cmd/meca/conversation.go.
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

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/commands/install"
	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	tuisession "github.com/keonho-kim/meca/internal/tui/session"
	"golang.org/x/term"
)

func main() {
	if handled, code := diagnostics.RunStderrRelay(os.Args); handled {
		os.Exit(code)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	output, err := diagnostics.OpenStderr()
	if err != nil {
		cancel()
		os.Exit(1)
	}
	level, levelErr := diagnostics.ParseLevel(os.Getenv("MECA_LOG_LEVEL"))
	sink, err := diagnostics.New(diagnostics.Options{Writer: output, Level: level, Redact: func(value string) string { return value }, Interrupt: output.Close})
	if err != nil {
		_ = output.Close()
		cancel()
		os.Exit(1)
	}
	ctx = diagnostics.WithSink(ctx, sink)
	err = levelErr
	if err == nil {
		err = faults.Guard(func() error { return start(ctx) })
	}
	code := diagnostics.Exit(ctx, output, err)
	if closeErr := sink.Close(); closeErr != nil && code == 0 {
		code = diagnostics.Exit(ctx, output, closeErr)
	}
	_ = output.Close()
	cancel()
	if code != 0 {
		os.Exit(code)
	}
}
func start(ctx context.Context) (err error) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return faults.Errorf(diagcatalog.MecaMECACurrentlySupportsLinuxAndMacOS)
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
		fmt.Fprintln(os.Stdout, "Usage: meca [path] [--print|--mode json] [--session ID] [prompt]\n       meca --session ID --export\n       meca --diagnose\n       meca [path] --server [--protocol sse|socket|grpc] [--listen IP:port]\n            [--session ID] [--openai] [--token-file PATH] [--tls-cert PATH --tls-key PATH]\n            [--allow-host host:port] [--allow-origin http(s)://host:port]\n       meca install [--no-shell]\n       meca resume | upgrade | uninstall")
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
	if mode.Action != "" {
		return maintenance(ctx, mode, home)
	}
	if err := startup.CheckAgentUser(os.Geteuid()); err != nil {
		return err
	}
	homeRoot, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, homeRoot.Close()) }()
	// This shared identity survives removal of the state directory, so startup
	// cannot race pending key cleanup or create replacement state during it.
	homeLease, err := statefs.Lease(homeRoot, false)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, homeLease.Close()) }()
	if err := install.CheckReady(home); err != nil {
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
	if err := install.CheckReady(home); err != nil {
		return err
	}
	configuration, err := bootstrap.Initialize(home)
	if err != nil {
		return faults.Errorf(diagcatalog.MecaBootstrapConfiguration, err)
	}
	state := filepath.Join(home, statefs.DirectoryName)
	ctx, closeLogs, err := loggingContext(ctx, state, mode)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, closeLogs()) }()
	if !mode.Export {
		stopRotation := (&credentials.Vault{State: state}).StartRotation(ctx)
		defer func() { err = errors.Join(err, stopRotation()) }()
	}
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		return faults.Errorf(diagcatalog.MecaOpenSessionDirectory, err)
	}
	if mode.Print {
		return runHeadless(ctx, mode, configuration, home, state, sessions)
	}
	if mode.Server {
		return runServer(ctx, mode, configuration, home, state, sessions)
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return faults.Errorf(diagcatalog.MecaUsePrintOrModeJsonWithoutAn)
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
