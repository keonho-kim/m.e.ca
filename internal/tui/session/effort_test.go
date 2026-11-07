// Purpose: Verify inline effort choices stay staged and save without extra model diagnostics.
// Pattern: Virtual terminal and persistence tests.
// Usage: go test ./internal/tui/session -run TestEffort.
// Related: internal/tui/session/model_form_keys.go, internal/tui/session/model_form_setup.go.
package session

import (
	"testing"

	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
)

func TestEffortSelectionPersistsOnlyAfterConfirmation(t *testing.T) {
	a := fixture(t)
	conn, err := connection.New(connection.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	a.connection.client, err = httpclient.NewClient(conn, "")
	if err != nil {
		t.Fatal(err)
	}
	low, off := "low", false
	a.connection.selection = connection.Selection{Connection: conn, Model: "alias", Thinking: &off, ThinkingSupported: true, ReasoningEffort: &low}
	a.openModel()
	f := a.connection.modelForm
	f.Row, f.Global = modelsettings.Effort, false
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	if f.Effort != "medium" || a.connection.selection.Effort() != "low" {
		t.Fatal("effort escaped the draft")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	finishModelChange(t, a)
	if a.page != "chat" || a.work.state.active() || a.connection.selection.Effort() != "medium" || *a.connection.selection.Thinking {
		t.Fatalf("saved=%+v status=%s", a.connection.selection, a.status)
	}
	stored, err := a.opts.Store.Selection(a.ctx, a.opts.Session.ID)
	if err != nil || stored.Effort() != "medium" {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
}

func TestEffortUnavailableCannotChange(t *testing.T) {
	a := fixture(t)
	conn, err := connection.New(connection.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	a.connection.selection = connection.Selection{Connection: conn, Model: "alias", Details: &connection.ModelInfo{ID: "alias", Reasoning: &connection.Reasoning{On: true, Off: true}}}
	a.openModel()
	a.connection.modelForm.Row = modelsettings.Effort
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	if a.page != "model" || a.connection.modelForm.Effort != "" || a.status == "" {
		t.Fatal("undeclared effort accepted")
	}
}
