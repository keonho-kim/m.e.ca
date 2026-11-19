// Purpose: Verify typed command queries filter visible candidates and deletion refreshes the search.
// Pattern: Terminal input and rendering regression test.
// Usage: go test ./internal/tui/session -run TestCommandSearch.
// Related: internal/commands/catalog.go, internal/tui/session/hints.go, internal/tui/session/render_hints.go.
package session

import (
	"slices"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/commands"
)

func TestCommandSearchFiltersAsYouType(t *testing.T) {
	a := screenFixture(t)
	typeInput(a, "!")
	initial := slices.Clone(a.compose.commandHints)
	if !slices.ContainsFunc(initial, func(hint commands.Suggestion) bool { return hint.Name == "!compact" }) {
		t.Fatalf("manual compaction missing from the command list: %v", initial)
	}
	typeInput(a, "odel")
	a.render()
	if len(a.compose.commandHints) != 1 || a.compose.commandHints[0].Name != "!model" {
		t.Fatalf("search=%v", a.compose.commandHints)
	}
	line := screenLine(a.screen, 20, 80)
	if !strings.Contains(line, "!model") {
		t.Fatalf("candidate row=%q", line)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "model" || a.compose.editor.String() != "" {
		t.Fatal("search selection did not execute canonical command")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	typeInput(a, "!unknown")
	if len(a.compose.commandHints) != 0 {
		t.Fatal("unmatched search returned commands")
	}
	for range "unknown" {
		a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace2, "", 0))
	}
	if a.compose.editor.String() != "!" || !slices.Equal(a.compose.commandHints, initial) {
		t.Fatal("deletion did not restore candidates")
	}
	typeInput(a, "모델")
	if len(a.compose.commandHints) != 1 || a.compose.commandHints[0].Name != "!model" {
		t.Fatalf("description search=%v", a.compose.commandHints)
	}
}
