// Purpose: Verify failure, foreground approval and shutdown respect queued-editor ownership.
// Pattern: Event-loop lifecycle contract tests.
// Usage: go test ./internal/tui/session -run 'TestPendingEditFailure|TestPendingEditApproval|TestPendingEditExit'.
// Related: internal/tui/session/pending_edit.go, internal/tui/session/steering.go, internal/tui/session/app.go.
package session

import (
	"context"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/agent/run"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestPendingEditFailurePreservesChangesAndUndeliveredSteering(t *testing.T) {
	for _, save := range []bool{false, true} {
		a := fixture(t)
		a.worker = workerAgent
		a.pending = []string{"first", "second"}
		a.steeringDrafts = []string{"undelivered direction"}
		a.editor.Set("unsent draft")
		pendingArrow(a, tcell.KeyLeft)
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " revised", 0))
		a.receive(notice{Done: true, Err: context.DeadlineExceeded})
		if a.worker.active() || a.pendingEdit == nil || a.editor.String() != "second revised" || len(a.steeringDrafts) != 0 {
			t.Fatal("failure overwrote queued editing with the normal input draft")
		}
		key, second := tcell.KeyEscape, "second"
		if save {
			key, second = tcell.KeyEnter, "second revised"
		}
		a.keyEvent(tcell.NewEventKey(key, "", 0))
		want := "first\n\n" + second + "\n\nundelivered direction\n\nunsent draft"
		if a.worker.active() || a.pendingEdit != nil || a.pending != nil || a.editor.String() != want {
			t.Fatalf("failed task resumed or lost input: %q; want %q", a.editor.String(), want)
		}
	}
}

func TestPendingEditApprovalKeepsForegroundKeys(t *testing.T) {
	a := fixture(t)
	a.worker = workerAgent
	a.pending = []string{"queued"}
	a.decision = make(chan bool, 1)
	pendingArrow(a, tcell.KeyLeft)
	a.receive(notice{Approval: &toolapi.Approval{Name: "protected operation"}})
	if labels := keymapLabels(t, a); labels["Esc"] != "거부" || labels["Ctrl/Cmd+Backspace"] != "" {
		t.Fatal(labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "chat" || a.pendingEdit == nil || a.editor.String() != "queued" {
		t.Fatal("approval confirmation replaced queued editing")
	}
	select {
	case approved := <-a.decision:
		if !approved {
			t.Fatal("foreground approval was rejected")
		}
	default:
		t.Fatal("Enter saved the queued edit instead of answering approval")
	}
}

func TestPendingEditExitDoesNotResumeDeferredDispatch(t *testing.T) {
	a := fixture(t)
	a.worker = workerAgent
	a.pending = []string{"queued"}
	a.editor.Set("unsent draft")
	pendingArrow(a, tcell.KeyLeft)
	a.receive(notice{Done: true, RunStatus: run.Answered})
	a.requestExit()
	if !a.exit || a.worker.active() || a.pendingEdit != nil || a.pending != nil || a.editor.String() != "queued\n\nunsent draft" {
		t.Fatal("shutdown started another request or retained queued editing")
	}
}
