// Purpose: Verify local UI input, approval and rendering through a virtual terminal.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session.
// Related: internal/tui/session/keys.go, internal/tui/session/render.go, internal/tui/session/tasks.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/policy"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
	"github.com/keonho-kim/meca/internal/tui/viewport"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) *app {
	t.Helper()
	home, root := t.TempDir(), t.TempDir()
	configuration, err := bootstrap.Initialize(home)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := sqlite.OpenSessions(filepath.Join(home, ".meca", "sessions"))
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
	vault := &credentials.Vault{State: filepath.Join(home, ".meca")}
	tools, err := local.NewExecutor(root, home, filepath.Join(home, ".meca"), vault.Redact)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tools.Close(); err != nil {
			t.Error(err)
		}
	})
	session := store.Session()
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
	a := &app{opts: Options{Home: home, State: filepath.Join(home, ".meca"), Config: configuration, Store: store, Session: session, Tools: tools, Vault: vault, LoadPrompts: func() (prompt.Templates, error) { return prompts.Load(filepath.Join(home, ".meca")) }, SaveConfig: func(c config.Config) error { return config.Save(filepath.Join(home, ".meca"), c) }}, ctx: ctx, cancel: cancel, screen: screen, page: "chat", settings: policy.Settings{Scope: policy.System}, output: viewport.View{Follow: true}, events: make(chan notice, 32)}
	t.Cleanup(a.shutdown)
	return a
}
func TestAtInputRemainsLiteral(t *testing.T) {
	a := fixture(t)
	typeInput(a, "read @src/main.go")
	if a.editor.String() != "read @src/main.go" || a.commandOpen || a.worker.active() {
		t.Fatal("@ triggered completion")
	}
}
func TestKeyInputPreservesGraphemeStrings(t *testing.T) {
	a := fixture(t)
	value := "한글 a\u0301 👨‍👩‍👧‍👦"
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, value, 0))
	if a.editor.String() != value {
		t.Fatalf("key event lost Unicode text: %q", a.editor.String())
	}
}
func TestRunConsumesNativeEventQueueAndExits(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	finished := make(chan error, 1)
	go func() { _, err := Run(ctx, a.screen, a.opts); finished <- err }()
	for _, event := range []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyRune, ":q", 0),
		tcell.NewEventKey(tcell.KeyEnter, "", 0),
	} {
		select {
		case a.screen.EventQ() <- event:
		case <-ctx.Done():
			if err := <-finished; err != nil {
				t.Fatal(err)
			}
			t.Fatal("event queue did not accept input")
		}
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != nil {
		t.Fatal("quit did not complete before cancellation")
	}
}

func TestInputSurvivesResizeAndModelMenu(t *testing.T) {
	a := fixture(t)
	a.selection.Model = "chosen"
	on := true
	a.selection.Thinking = &on
	a.editor.Set("draft")
	a.render()
	line := screenLine(a.screen, 24-a.footerRows, 80)
	if strings.TrimSpace(line) != "chosen" {
		t.Fatal(line)
	}
	off := false
	a.selection.Thinking = &off
	a.render()
	if strings.Contains(screenLine(a.screen, 24-a.footerRows, 80), "추론") {
		t.Fatal("OFF is visible")
	}
	a.openModel()
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.editor.String() != "draft" {
		t.Fatal("menu cleared draft")
	}
	a.screen.SetSize(32, 10)
	a.render()
	if a.editor.String() != "draft" {
		t.Fatal("resize cleared draft")
	}
}
func screenLine(screen tcell.Screen, y, width int) string {
	var b strings.Builder
	for x := 0; x < width; x++ {
		r, _, cells := screen.Get(x, y)
		b.WriteString(r)
		if cells > 1 {
			x += cells - 1
		}
	}
	return b.String()
}
func TestStatusModelLabels(t *testing.T) {
	for _, test := range []struct {
		name     string
		thinking bool
		want     string
	}{
		{"model", false, "chosen"},
		{"thinking", true, "chosen"},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := fixture(t)
			a.selection.Model = "chosen"
			a.selection.Thinking = &test.thinking
			a.render()
			if line := strings.TrimSpace(screenLine(a.screen, 24-a.footerRows, 80)); line != test.want {
				t.Fatalf("status = %q, want %q", line, test.want)
			}
		})
	}
}
func TestCancellingConnectionSelectionPreservesCurrentModel(t *testing.T) {
	a := fixture(t)
	a.selection.Model = "current-model"
	candidate := modelconn.Selection{Connection: modelconn.Connection{ID: "candidate"}}
	a.receive(notice{Done: true, Selection: &candidate, Models: []string{"new-model"}})
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.selection.Model != "current-model" {
		t.Fatal("connection menu cleared the current model")
	}
}
