// Purpose: Verify typed command queries filter visible candidates and deletion refreshes the search.
// Pattern: Terminal input and rendering regression test.
// Usage: go test ./internal/tui/session -run TestCommandSearch.
// Related: internal/commands/catalog.go, internal/tui/session/hints.go, internal/tui/session/render_hints.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
)

func TestCommandSearchFiltersAsYouType(t *testing.T) {
	a := fixture(t)
	typeInput(a, "!")
	if len(a.commandHints) != 6 {
		t.Fatal("initial command list missing")
	}
	typeInput(a, "odel")
	a.render()
	if len(a.commandHints) != 1 || a.commandHints[0].Name != "!model" {
		t.Fatalf("search=%v", a.commandHints)
	}
	line := screenLine(a.screen, 20, 80)
	if !strings.Contains(line, "!model") {
		t.Fatalf("candidate row=%q", line)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "model" || a.editor.String() != "" {
		t.Fatal("search selection did not execute canonical command")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	typeInput(a, "!unknown")
	if len(a.commandHints) != 0 {
		t.Fatal("unmatched search returned commands")
	}
	for range "unknown" {
		a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace2, "", 0))
	}
	if a.editor.String() != "!" || len(a.commandHints) != 6 {
		t.Fatal("deletion did not restore candidates")
	}
	typeInput(a, "모델")
	if len(a.commandHints) != 1 || a.commandHints[0].Name != "!model" {
		t.Fatalf("description search=%v", a.commandHints)
	}
}
