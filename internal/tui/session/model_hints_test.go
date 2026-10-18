// Purpose: Verify model-ID completion filters below the field and fills without connecting.
// Pattern: Terminal input workflow tests.
// Usage: go test ./internal/tui/session -run TestModelHints.
// Related: internal/tui/session/model_hints.go, internal/tui/view/menu.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"strings"
	"testing"
)

func TestModelHintsFilterSelectAndClose(t *testing.T) {
	a := fixture(t)
	a.openModelInput([]string{"Qwen3.8-27B", "Qwen-3.8-180B", "Gemma-4-31B"})
	a.render()
	if !strings.Contains(screenLine(a.screen, 3, 80), "Qwen3.8-27B") {
		t.Fatal("candidates not below field")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "3.8", 0))
	if len(a.modelHints) != 2 {
		t.Fatal(a.modelHints)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.menuInput.String() != "Qwen-3.8-180B" || a.modelHintsOpen || a.busy || a.page != "manual" {
		t.Fatal("Enter did not fill without connecting")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "GEMMA", 0))
	if len(a.modelHints) != 1 || a.modelHints[0] != "Gemma-4-31B" {
		t.Fatal("case-insensitive matching failed")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "manual" || a.menuInput.String() != "GEMMA" || a.modelHintsOpen {
		t.Fatal("Esc lost input")
	}
}

func TestModelHintsNoMatchPasteAndResize(t *testing.T) {
	a := fixture(t)
	a.openModelInput([]string{"one", "two", "three", "four", "five", "six", "seven", "eight"})
	a.screen.SetSize(24, 10)
	for i := 0; i < 7; i++ {
		a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	}
	a.render()
	visible := ""
	for row := 3; row < 8; row++ {
		visible += screenLine(a.screen, row, 24)
	}
	if !strings.Contains(visible, "eight") {
		t.Fatal("selected candidate hidden in small terminal")
	}
	a.pasteBoundary(true)
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "custom/model", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.pasteBoundary(false)
	if a.busy || a.menuInput.String() != "custom/model" || len(a.modelHints) != 0 {
		t.Fatal("paste executed or failed to filter")
	}
	a.render()
	if !strings.Contains(screenLine(a.screen, 3, 24), "No matches") {
		t.Fatal("missing no-match indication")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if a.menuInput.String() != "custom/model" {
		t.Fatal("Tab changed unmatched ID")
	}
}
