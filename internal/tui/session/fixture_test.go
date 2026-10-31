// Purpose: Supply screen-only fixtures and isolated persistence fixtures to terminal behavior tests.
// Pattern: Layered test setup at I/O boundaries.
// Usage: Rendering and input tests use screenFixture; request and restore tests use fixture.
// Related: internal/tui/session/app_test.go, internal/tui/session/state.go, internal/tui/session/app.go.
package session

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/policy"
	storagesession "github.com/keonho-kim/meca/internal/storage/session"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func screenFixture(t testing.TB) *app {
	t.Helper()
	home, root := t.TempDir(), t.TempDir()
	state := filepath.Join(home, ".meca")
	a := terminalFixture(t, Options{
		Home: home, State: state, Config: config.Defaults(),
		Session: storagesession.Session{ID: "screen-test", ProjectRoot: root, DisplayRoot: root},
		Vault:   &credentials.Vault{State: state},
	})
	a.historyEnd = true
	return a
}

func fixture(t testing.TB) *app {
	t.Helper()
	home, root := t.TempDir(), t.TempDir()
	state := filepath.Join(home, ".meca")
	configuration, err := bootstrap.Initialize(home)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := sessions.Create(context.Background(), root, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	vault := &credentials.Vault{State: state}
	tools, err := local.NewExecutor(root, home, state, vault.Redact)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tools.Close(); err != nil {
			t.Error(err)
		}
	})
	return terminalFixture(t, Options{
		Home: home, State: state, Config: configuration,
		Store: store, Session: store.Session(), Tools: tools, Files: tools.Files, Vault: vault,
		LoadPrompts: func() (prompt.Templates, error) { return prompts.Load(state) },
		SaveConfig:  func(c config.Config) error { return config.Save(state, c) },
	})
}

func terminalFixture(t testing.TB, opts Options) *app {
	t.Helper()
	terminal := vt.NewMockTerm(vt.MockOptSize{X: 80, Y: 24})
	screen, err := tcell.NewTerminfoScreenFromTty(terminal)
	if err != nil {
		t.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	screen.SetSize(80, 24)
	t.Cleanup(screen.Fini)
	ctx, cancel := context.WithCancel(context.Background())
	a := &app{
		opts: opts, ctx: ctx, cancel: cancel, screen: screen, page: "chat",
		settings: policy.Settings{Scope: policy.System}, output: viewport.View{Follow: true},
		events: make(chan notice, 32),
	}
	t.Cleanup(a.shutdown)
	return a
}
