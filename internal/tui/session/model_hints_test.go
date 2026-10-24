// Purpose: Verify model-field partial search, candidate filling and literal paste within the settings page.
// Pattern: Virtual terminal editing tests.
// Usage: go test ./internal/tui/session -run TestModelHints.
// Related: internal/tui/session/model_form_keys.go, internal/tui/session/paste.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"testing"
)

func TestModelHintsFilterSelectAndClose(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	f.row, f.editing = modelName, true
	f.models = []string{"Qwen3.8-27B", "Qwen-3.8-180B", "Gemma-4-31B"}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "3.8", 0))
	if len(f.hints) != 2 {
		t.Fatal(f.hints)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if f.name.String() != "Qwen-3.8-180B" || f.editing || a.worker.active() || a.selection.Model != "" {
		t.Fatal("Tab did not only fill the model field")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if !f.confirm || a.worker.active() {
		t.Fatal("Enter should ask to save rather than connect")
	}
}

func TestModelHintsPasteDoesNotSave(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	f.row, f.editing = modelName, true
	f.models = []string{"one", "two"}
	a.pasteBoundary(true)
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "custom/model", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.pasteBoundary(false)
	if a.worker.active() || f.confirm || f.name.String() != "custom/model" || len(f.hints) != 0 {
		t.Fatal("paste submitted settings")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "model" || f.editing || f.name.String() != "custom/model" {
		t.Fatal("Esc lost draft model")
	}
}
