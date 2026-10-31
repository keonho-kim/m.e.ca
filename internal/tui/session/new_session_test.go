// Purpose: Verify !new requests a clean session transition and joins active workers before returning.
// Pattern: Terminal lifecycle regression test.
// Usage: go test ./internal/tui/session -run TestNewSession.
// Related: internal/tui/session/app.go, cmd/meca/conversation.go.
package session

import (
	"context"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
)

func TestNewSessionCommandCancelsActiveWork(t *testing.T) {
	a := screenFixture(t)
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.worker, a.taskCancel = workerSetup, cancel
	a.editor.Set("!new")
	a.submit()
	if !a.newSession || !a.exit || ctx.Err() == nil {
		t.Fatal("new session did not cancel the active session")
	}
}

func TestNewSessionReturnsTransitionFromTerminal(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := make(chan Action, 1)
	go func() {
		action, err := Run(ctx, a.screen, a.opts)
		if err != nil {
			t.Error(err)
		}
		result <- action
	}()
	a.screen.EventQ() <- tcell.NewEventKey(tcell.KeyRune, "!new", 0)
	a.screen.EventQ() <- tcell.NewEventKey(tcell.KeyEnter, "", 0)
	select {
	case action := <-result:
		if action != NewSession {
			t.Fatalf("action=%v", action)
		}
	case <-ctx.Done():
		t.Fatal("new-session request did not return")
	}
}
