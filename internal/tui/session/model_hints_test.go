// Purpose: Verify model-field partial search, candidate filling and literal paste within the settings page.
// Pattern: Virtual terminal editing tests.
// Usage: go test ./internal/tui/session -run TestModelHints.
// Related: internal/tui/session/model_form_keys.go, internal/tui/session/paste.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
	"testing"
)

func TestModelHintsFilterSelectAndClose(t *testing.T) {
	a := screenFixture(t)
	a.openModel()
	f := a.modelForm
	f.Row, f.Editing = modelsettings.Name, true
	f.Models = []string{"Qwen3.8-27B", "Qwen-3.8-180B", "Gemma-4-31B"}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "3.8", 0))
	if len(f.Hints) != 2 {
		t.Fatal(f.Hints)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if f.Name.String() != "Qwen-3.8-180B" || f.Editing || a.worker.active() || a.selection.Model != "" {
		t.Fatal("Tab did not only fill the model field")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if !f.Confirm || a.worker.active() {
		t.Fatal("Enter should ask to save rather than connect")
	}
}

func TestModelHintsPasteDoesNotSave(t *testing.T) {
	a := screenFixture(t)
	a.openModel()
	f := a.modelForm
	f.Row, f.Editing = modelsettings.Name, true
	f.Models = []string{"one", "two"}
	a.pasteBoundary(true)
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "custom/model", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.pasteBoundary(false)
	if a.worker.active() || f.Confirm || f.Name.String() != "custom/model" || len(f.Hints) != 0 {
		t.Fatal("paste submitted settings")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "model" || f.Editing || f.Name.String() != "custom/model" {
		t.Fatal("Esc lost draft model")
	}
}
