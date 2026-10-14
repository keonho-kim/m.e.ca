// Purpose: Verify explicit app exit cancels work from every screen without approving a Plan.
// Pattern: Terminal lifecycle tests.
// Usage: go test ./internal/tui/session -run 'TestExit|TestRunExit'.
// Related: internal/tui/session/keys.go, internal/tui/session/app.go, internal/commands/catalog.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"testing"
	"time"
)

func TestExitKeyCancelsFromEveryScreen(t *testing.T) {
	for _, page := range []string{"chat", "files", "model", "approval", "planapproval"} {
		t.Run(page, func(t *testing.T) {
			a := fixture(t)
			a.page, a.busy, a.planMode = page, true, true
			taskCtx, cancel := context.WithCancel(a.ctx)
			defer cancel()
			a.taskCancel = cancel
			a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlC, "", 0))
			if !a.exit || taskCtx.Err() == nil || !a.planMode {
				t.Fatal("exit did not cancel work or changed unapproved Plan")
			}
		})
	}
}

func TestExitCommandWorksWhileBusyAndInMenus(t *testing.T) {
	for _, page := range []string{"chat", "files", "model", "endpoint", "approval", "planapproval"} {
		t.Run(page, func(t *testing.T) {
			a := fixture(t)
			a.page, a.busy, a.planMode = page, true, true
			if page == "chat" {
				a.editor.Set("!quit")
			} else {
				a.menuInput.Set("!quit")
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
			if !a.exit {
				t.Fatal("!quit did not exit")
			}
		})
	}
	a := fixture(t)
	a.page = "files"
	for _, r := range "!quit" {
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, string(r), 0))
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if !a.exit {
		t.Fatal("browser focus could not type !quit")
	}
}

func TestRunExitEndsTerminalLoop(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- Run(ctx, a.screen, a.opts) }()
	a.screen.EventQ() <- tcell.NewEventKey(tcell.KeyCtrlC, "", 0)
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("Ctrl+C did not end terminal loop")
	}
}

func TestShutdownJoinsBlockedWorker(t *testing.T) {
	a := fixture(t)
	a.ctx, a.cancel = context.WithCancel(context.Background())
	for i := 0; i < cap(a.events); i++ {
		a.events <- notice{}
	}
	started := make(chan struct{})
	stopped := make(chan error, 1)
	a.workers.Add(1)
	go func() { defer a.workers.Done(); close(started); stopped <- a.post(a.ctx, notice{Done: true}) }()
	<-started
	a.shutdown()
	if err := <-stopped; err != context.Canceled {
		t.Fatalf("queued worker not canceled: %v", err)
	}
	if !a.exit || len(a.events) != 0 {
		t.Fatal("shutdown did not drain pending ownership")
	}
}
