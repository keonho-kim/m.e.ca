// Purpose: Verify cancelled and superseded connection results cannot reopen menus or replace the active model.
// Pattern: Worker lifecycle contract tests.
// Usage: go test ./internal/tui/session -run TestCancelledSetup.
// Related: internal/tui/session/setup_cancel.go, internal/tui/session/worker_result.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"testing"
)

func TestCancelledSetupDiscardsSuccessfulLateResult(t *testing.T) {
	a := fixture(t)
	a.worker = workerSetup
	a.setupRevision = 2
	a.selection.Model = "current"
	cancelled := false
	a.taskCancel = func() { cancelled = true }
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if !cancelled {
		t.Fatal("connection not cancelled")
	}
	a.receive(notice{SetupRevision: 2, Done: true, Models: []string{"late"}, Selection: &connection.Selection{}})
	if a.worker.active() || a.page != "chat" || a.selection.Model != "current" || a.modelForm != nil {
		t.Fatal("late result changed cancelled screen")
	}
	a.worker = workerSetup
	a.setupRevision = 3
	a.receive(notice{SetupRevision: 2, Done: true, Models: []string{"old"}})
	if !a.worker.active() || a.page != "chat" {
		t.Fatal("old completion stopped newer request")
	}
}
