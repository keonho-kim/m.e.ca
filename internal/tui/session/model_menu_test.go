// Purpose: Verify model-field editing, concrete server defaults and staged Thinking changes.
// Pattern: Virtual terminal behavior tests.
// Usage: go test ./internal/tui/session -run 'TestEndpoint|TestModelThinking'.
// Related: internal/tui/session/model_form_keys.go, internal/tui/session/render_model.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tui/locale"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
	"strings"
	"testing"
)

func TestEndpointClearRemovesSuffixAndRejectsEmpty(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	f.Row, f.Editing = modelsettings.Endpoint, true
	f.Endpoint.Set("http://localhost:8000/v1")
	for i := 0; i < 3; i++ {
		a.keyEvent(tcell.NewEventKey(tcell.KeyLeft, "", 0))
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, "", 0))
	if f.Endpoint.String() != "" {
		t.Fatal("endpoint suffix survived clearing")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	if a.page != "model" || a.status == "" {
		t.Fatal("empty endpoint was accepted")
	}
	f.Editing = true
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "http://127.0.0.1:9000", 0))
	if f.Endpoint.String() != "http://127.0.0.1:9000" {
		t.Fatal(f.Endpoint.String())
	}
}

func TestEndpointCursorTracksEditingAndHorizontalScroll(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	f.Row, f.Editing = modelsettings.Endpoint, true
	f.Endpoint.Set("http://localhost:8000/v1")
	a.keyEvent(tcell.NewEventKey(tcell.KeyLeft, "", 0))
	a.render()
	tty, ok := a.screen.Tty()
	if !ok {
		t.Fatal("missing terminal")
	}
	terminal := tty.(vt.MockTerm)
	if err := terminal.Drain(); err != nil {
		t.Fatal(err)
	}
	if int(terminal.Pos().X) != a.inputRegion.x+f.Endpoint.Cursor {
		t.Fatal("cursor differs from field")
	}
	a.screen.SetSize(20, 10)
	f.Endpoint.End()
	a.render()
	if err := terminal.Drain(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(screenLine(a.screen, a.inputRegion.y, 20), "8000/v1") || terminal.Pos().X != 19 {
		t.Fatal("long endpoint tail hidden")
	}
}

func TestModelThinkingUsesKnownDefaultAndStagesChanges(t *testing.T) {
	for _, known := range []bool{false, true} {
		a := fixture(t)
		a.language = locale.English
		conn, err := connection.New(connection.Ollama, "")
		if err != nil {
			t.Fatal(err)
		}
		on := true
		r := &connection.Reasoning{On: true, Off: true}
		if known {
			r.DefaultOn = &on
		}
		a.selection = connection.Selection{Connection: conn, Model: "alias", Details: &connection.ModelInfo{ID: "alias", Reasoning: r}, ThinkingSupported: true}
		a.openModel()
		f := a.modelForm
		line := screenLineAfterModelRender(a, 2+modelsettings.Thinking)
		if known && (strings.Contains(line, "DEFAULT") || !strings.Contains(line, "ON")) {
			t.Fatal(line)
		}
		if !known && !strings.Contains(line, "DEFAULT") {
			t.Fatal(line)
		}
		f.Row = modelsettings.Thinking
		a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
		if f.Thinking == nil || a.selection.Thinking != nil {
			t.Fatal("draft did not remain staged")
		}
	}
}

func screenLineAfterModelRender(a *app, row int) string {
	a.render()
	return screenLine(a.screen, row, 80)
}
