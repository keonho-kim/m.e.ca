// Purpose: Verify the single-screen model settings and explicit save confirmation keyboard contract.
// Pattern: Virtual terminal behavior tests.
// Usage: go test ./internal/tui/session -run TestModelDashboard.
// Related: internal/tui/session/model_form.go, internal/tui/session/model_form_keys.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func modelScreen(a *app) string {
	a.render()
	w, h := a.screen.Size()
	var body strings.Builder
	for y := 0; y < h; y++ {
		body.WriteString(screenLine(a.screen, y, w))
		body.WriteByte('\n')
	}
	return body.String()
}

func TestModelDashboardShowsAllSettings(t *testing.T) {
	a := fixture(t)
	a.language = locale.English
	a.openModel()
	body := modelScreen(a)
	for _, label := range []string{"Provider", "Base URL", "API Key", "Model ID", "Thinking", "Thinking effort", "Save destination", "Connection diagnostics"} {
		if !strings.Contains(body, label) {
			t.Errorf("missing setting %q: %s", label, body)
		}
	}
}

func TestModelDashboardEnterAsksBeforeSaving(t *testing.T) {
	a := fixture(t)
	a.language = locale.English
	a.openModel()
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if body := modelScreen(a); !strings.Contains(body, "Save current settings?") || !strings.Contains(body, "y") || !strings.Contains(body, "n") || !strings.Contains(body, "Esc") {
		t.Fatalf("save confirmation missing: %s", body)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "model" || strings.Contains(modelScreen(a), "Save current settings?") {
		t.Fatal("Esc did not return to settings")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "n", 0))
	if a.page != "chat" || a.selection.Model != "" {
		t.Fatal("n did not discard and return to chat")
	}
}

func TestModelDashboardKeymapMatchesFocusedRow(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	f.row = modelEndpoint
	if labels := keymapLabels(t, a); labels["→"] != "입력" || labels["←"] != "" {
		t.Fatal(labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	if !f.editing {
		t.Fatal("right did not enter the text field")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	if f.row != modelCredential || f.editing {
		t.Fatal("down did not move to the next setting")
	}
	f.row = modelThinking
	if labels := keymapLabels(t, a); labels["→"] != "" {
		t.Fatal("unknown capabilities advertised an executable toggle")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if labels := keymapLabels(t, a); labels["y"] == "" || labels["n"] == "" || labels["Esc"] != "설정 계속" {
		t.Fatal(labels)
	}
}
