// Purpose: Verify localized labels retain bold concrete ON/OFF selection on the settings screen.
// Pattern: Bilingual virtual terminal test.
// Usage: go test ./internal/tui/session -run TestLocalizedThinking.
// Related: internal/tui/session/render_model.go, internal/tui/view/settings.go.
package session

import (
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tui/locale"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
	"strings"
	"testing"
)

func TestLocalizedThinkingChoicePositions(t *testing.T) {
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		a := screenFixture(t)
		a.language = lang
		conn, err := connection.New(connection.VLLM, "")
		if err != nil {
			t.Fatal(err)
		}
		on := true
		a.selection = connection.Selection{Connection: conn, Model: "alias", Thinking: &on, ThinkingSupported: true}
		a.openModel()
		a.render()
		y := 2 + modelsettings.Thinking
		shown := screenLine(a.screen, y, 80)
		if !strings.Contains(shown, lang.Text("Thinking")) || !strings.Contains(shown, "ON / OFF") || strings.Contains(shown, "DEFAULT") {
			t.Fatal(shown)
		}
		_, selected, _ := a.screen.Get(25, y)
		_, other, _ := a.screen.Get(30, y)
		if !selected.HasBold() || !other.HasDim() {
			t.Fatal("concrete selection emphasis lost")
		}
	}
}
