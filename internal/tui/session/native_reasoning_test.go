// Purpose: Verify mandatory reasoning remains visible while OFF is unavailable and effort remains selectable.
// Pattern: Terminal behavior test.
// Usage: go test ./internal/tui/session -run TestNativeReasoningMenu.
// Related: internal/tui/session/effort.go, internal/tui/session/render_menu.go, internal/tui/view/menu.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	selectionpolicy "github.com/keonho-kim/v-code/internal/provider/selection"
	"strings"
	"testing"
)

func TestNativeReasoningMenu(t *testing.T) {
	a := fixture(t)
	on := true
	a.selection = connection.Selection{Model: "alias", Thinking: &on, Details: &connection.ModelInfo{ID: "alias", Reasoning: &connection.Reasoning{Levels: []string{"Light", "Deep"}, Default: "Light"}}}
	a.openModel()
	a.render()
	line := screenLine(a.screen, 3, 80)
	if !strings.Contains(line, "켜짐") || !strings.Contains(line, "끄기 불가") {
		t.Fatalf("mandatory reasoning unclear: %s", line)
	}
	_, style, _ := a.screen.Get(5, 3)
	if !style.HasBold() {
		t.Fatal("active mandatory ON not emphasized")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "model" || !*a.selection.Thinking {
		t.Fatal("unsupported OFF became selectable")
	}
	a.openEffort()
	if a.page != "effort" || len(a.choices) != 3 || selectionpolicy.EffortValues(a.selection)[1] != "Deep" {
		t.Fatal("native strength choices lost")
	}
}

func TestNativeOnOnlyEffortFromServerDefault(t *testing.T) {
	a := fixture(t)
	conn, err := connection.New(connection.Ollama, "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = connection.Selection{Connection: conn, Model: "alias", Details: &connection.ModelInfo{ID: "alias", Reasoning: &connection.Reasoning{Levels: []string{"Light", "Deep"}, Default: "Light"}}}
	if err := a.applyEffort("Deep"); err != nil {
		t.Fatal(err)
	}
	if a.selection.Thinking == nil || !*a.selection.Thinking || a.selection.Effort() != "Deep" {
		t.Fatalf("selection=%+v", a.selection)
	}
}
