// Purpose: Verify one deferred input stays ordered and returns to the draft on interruption.
// Pattern: Terminal state transition tests.
// Usage: go test ./internal/tui/session -run TestPending.
// Related: internal/tui/session/pending.go, internal/tui/session/keys.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"testing"
)

func TestPendingInputPreservesOrderAndDraft(t *testing.T) {
	a := fixture(t)
	a.busy, a.agentActive = true, true
	a.editor.Set("next request")
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if a.pending != "next request" || a.editor.String() != "" || !a.busy {
		t.Fatal("Tab did not reserve input")
	}
	a.editor.Set("later draft")
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if a.pending != "next request" || a.editor.String() != "later draft" {
		t.Fatal("second Tab overwrote input")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.taskCancel = cancel
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if ctx.Err() != context.Canceled || a.pending != "" || a.editor.String() != "next request\n\nlater draft" {
		t.Fatal("Esc lost pending input")
	}
}

func TestPendingRejectsLocalCommands(t *testing.T) {
	a := fixture(t)
	a.busy, a.agentActive = true, true
	a.editor.Set("!quit")
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if a.pending != "" || a.exit || a.editor.String() != "!quit" {
		t.Fatal("local command was queued or executed")
	}
}

func TestPendingFailureRestoresDraftWithoutStarting(t *testing.T) {
	a := fixture(t)
	a.busy, a.agentActive = true, true
	a.pending = "next request"
	a.editor.Set("unsent draft")
	a.receive(notice{Done: true, Err: context.DeadlineExceeded})
	if a.busy || a.pending != "" || a.editor.String() != "next request\n\nunsent draft" {
		t.Fatal("failed worker started or lost pending request")
	}
}
