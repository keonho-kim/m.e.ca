// Purpose: Verify local UI input, approval, Plan gating and rendering through a virtual terminal.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session.
// Related: internal/tui/session/keys.go, internal/tui/session/render.go, internal/tui/session/tasks.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/config/bootstrap"
	"github.com/keonho-kim/v-code/internal/config/credentials"
	"github.com/keonho-kim/v-code/internal/config/prompts"
	"github.com/keonho-kim/v-code/internal/policy"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"github.com/keonho-kim/v-code/internal/tools/local"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
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
	sessions, err := sqlite.OpenSessions(filepath.Join(home, ".v", "sessions"))
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
	vault := &credentials.Vault{}
	tools, err := local.NewExecutor(root, home, filepath.Join(home, ".v"), vault.Redact)
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
	return &app{opts: Options{Home: home, State: filepath.Join(home, ".v"), Config: configuration, Store: store, Session: session, Tools: tools, Vault: vault, LoadPrompts: func() (prompt.Templates, error) { return prompts.Load(filepath.Join(home, ".v")) }, SaveConfig: func(c config.Config) error { return config.Save(filepath.Join(home, ".v"), c) }}, ctx: context.Background(), screen: screen, page: "chat", settings: configuration.Defaults, output: viewport.View{Follow: true}, events: make(chan notice, 32)}
}
func TestAtInputRemainsLiteral(t *testing.T) {
	a := fixture(t)
	typeInput(a, "read @src/main.go")
	if a.editor.String() != "read @src/main.go" || a.commandOpen || a.busy {
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
func TestPlanAndApprovalCannotBeBypassed(t *testing.T) {
	a := fixture(t)
	a.editor.Set("!plan")
	a.submit()
	if !a.planMode || a.plan.Status != planning.Planning {
		t.Fatal("did not enter Plan")
	}
	a.editor.Set(":q")
	a.submit()
	if a.exit || !a.planMode {
		t.Fatal("quit bypassed Plan")
	}
	a.decision = make(chan bool, 1)
	a.proposal = &toolapi.Approval{Name: "Editing", Target: "auth.go", Preview: "old -> new"}
	a.page = "approval"
	a.choices = []string{"Approve", "Reject"}
	a.choice = 1
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if <-a.decision {
		t.Fatal("reject became approval")
	}
}
func TestThinkingOffAndInputSurviveResizeAndMenu(t *testing.T) {
	a := fixture(t)
	a.selection.Model = "chosen"
	on := true
	a.selection.Thinking = &on
	a.editor.Set("draft")
	a.render()
	line := screenLine(a.screen, 22, 80)
	if !strings.Contains(line, "chosen Thinking") {
		t.Fatal(line)
	}
	off := false
	a.selection.Thinking = &off
	a.render()
	if strings.Contains(screenLine(a.screen, 22, 80), "Thinking") {
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
func TestStatusPolicyLabels(t *testing.T) {
	for _, test := range []struct {
		name       string
		permission policy.Permission
		scope      policy.Scope
		thinking   bool
		plan       bool
		want       string
	}{
		{"default", policy.Approve, policy.Directory, false, false, "chosen │ 전체 시스템"},
		{"system", policy.Approve, policy.System, false, false, "chosen │ 전체 시스템"},
		{"automatic", policy.AllowAll, policy.Directory, false, false, "chosen │ 자동 승인 │ 전체 시스템"},
		{"thinking and plan", policy.AllowAll, policy.System, true, true, "chosen Thinking │ 자동 승인 │ 전체 시스템 │ PLAN"},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := fixture(t)
			a.selection.Model = "chosen"
			a.selection.Thinking = &test.thinking
			a.settings = policy.Settings{Permission: test.permission, Scope: test.scope}
			a.planMode = test.plan
			a.render()
			if line := strings.TrimSpace(screenLine(a.screen, 22, 80)); !strings.HasPrefix(line, test.want) {
				t.Fatalf("status = %q, want %q", line, test.want)
			}
		})
	}
}
func TestPolicyChoiceAlwaysAsksGlobalAndPreservesOverrides(t *testing.T) {
	a := fixture(t)
	a.page = "permission"
	a.choices = []string{"approve", "allow-all"}
	a.choice = 0
	a.choose()
	if a.page != "global" || a.settings.Permission != policy.AllowAll {
		t.Fatal("permission applied before destination choice")
	}
	a.choice = 0
	a.choose()
	if a.settings.Permission != policy.Approve || a.settings.Scope != policy.System {
		t.Fatal("session permission missing or scope changed")
	}
	a.changeField = "permission"
	a.changeValue = "allow-all"
	a.applySetting(true)
	if a.settings.Permission != policy.Approve {
		t.Fatal("global replaced session override")
	}
}
func TestRestoredUnapprovedPlanRemainsGated(t *testing.T) {
	a := fixture(t)
	p, err := planning.NewPlan("p", a.opts.Session.ID, "fix", []string{"read"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err = a.opts.Store.SavePlan(a.ctx, p); err != nil {
		t.Fatal(err)
	}
	if err = a.restore(); err != nil {
		t.Fatal(err)
	}
	if !a.planMode {
		t.Fatal("lost unapproved Plan")
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

func closeTestStore(t *testing.T, store Store) error {
	t.Helper()
	closer, ok := store.(interface{ Close() error })
	if !ok {
		t.Fatal("fixture repository has no close boundary")
	}
	return closer.Close()
}
