// Purpose: Verify LAT toggles locally, appears in the footer and cannot alter an active request.
// Pattern: Virtual terminal behavior tests.
// Usage: go test ./internal/tui/session -run TestLAT.
// Related: internal/tui/session/commands.go, internal/tui/view/header.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestLATCommandTogglesAndDisplaysCurrentMode(t *testing.T) {
	for _, language := range []locale.Language{locale.Korean, locale.English} {
		a := screenFixture(t)
		a.language = language
		a.selection.Model = "test-model"
		if a.lat {
			t.Fatal("LAT should start off")
		}
		for _, enabled := range []bool{true, false} {
			typeInput(a, "!lat")
			if len(a.commandHints) != 1 || a.commandHints[0].Name != "!lat" {
				t.Fatal("LAT completion missing")
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
			a.render()
			footer := screenLine(a.screen, 24-a.footerRows, 80)
			if a.lat != enabled || strings.Contains(footer, "LAT") != enabled || a.editor.String() != "" || a.worker.active() {
				t.Fatalf("toggle=%t footer=%q", a.lat, footer)
			}
		}
	}
}

func TestLATCannotChangeAnActiveRequest(t *testing.T) {
	a := screenFixture(t)
	a.worker = workerAgent
	typeInput(a, "!lat")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.lat || a.editor.String() != "!lat" {
		t.Fatal("active request mode changed or draft lost")
	}
}
