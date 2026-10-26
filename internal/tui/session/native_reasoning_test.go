// Purpose: Verify mandatory reasoning shows ON and effort choices preserve server literals.
// Pattern: Virtual terminal settings tests.
// Usage: go test ./internal/tui/session -run TestNativeReasoning.
// Related: internal/tui/session/model_form_keys.go, internal/tui/session/render_model.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/provider/connection"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
	"strings"
	"testing"
)

func TestNativeReasoningMenu(t *testing.T) {
	a := fixture(t)
	conn, err := connection.New(connection.Ollama, "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = connection.Selection{Connection: conn, Model: "alias", Details: &connection.ModelInfo{ID: "alias", Reasoning: &connection.Reasoning{Levels: []string{"Light", "Deep"}, Default: "Light"}}}
	a.openModel()
	f := a.modelForm
	line := screenLineAfterModelRender(a, 2+modelsettings.Thinking)
	if strings.Contains(line, "DEFAULT") || !strings.Contains(line, "ON") || !strings.Contains(line, "끄기 불가") {
		t.Fatal(line)
	}
	f.Row = modelsettings.Thinking
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	if f.Thinking == nil || !*f.Thinking {
		t.Fatal("mandatory ON became OFF")
	}
	f.Row = modelsettings.Effort
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	if f.Effort != "Deep" || a.selection.Effort() != "" {
		t.Fatal("literal native effort lost or persisted early")
	}
}
