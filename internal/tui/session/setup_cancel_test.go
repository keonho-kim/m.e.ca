// Purpose: Verify cancelled and superseded connection results cannot reopen menus or replace the active model.
// Pattern: Worker lifecycle contract tests.
// Usage: go test ./internal/tui/session -run TestCancelledSetup.
// Related: internal/tui/session/setup_cancel.go, internal/tui/session/worker_result.go.
package session

import (
	"testing"

	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestCancelledSetupDiscardsSuccessfulLateResult(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerSetup
	a.connection.setupRevision = 2
	a.connection.selection.Model = "current"
	cancelled := false
	a.work.cancel = func() { cancelled = true }
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if !cancelled {
		t.Fatal("connection not cancelled")
	}
	a.receive(notice{SetupRevision: 2, Done: true, Models: []string{"late"}, Selection: &connection.Selection{}})
	if a.work.state.active() || a.page != "chat" || a.connection.selection.Model != "current" || a.connection.modelForm != nil {
		t.Fatal("late result changed cancelled screen")
	}
	a.work.state = workerSetup
	a.connection.setupRevision = 3
	a.receive(notice{SetupRevision: 2, Done: true, Models: []string{"old"}})
	if !a.work.state.active() || a.page != "chat" {
		t.Fatal("old completion stopped newer request")
	}
}
