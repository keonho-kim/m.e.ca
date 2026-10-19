// Purpose: Verify translated Thinking choices retain emphasis and cell positions in both UI languages.
// Pattern: Bilingual terminal interaction tests.
// Usage: go test ./internal/tui/session -run TestLocalizedThinking.
// Related: internal/tui/view/thinking.go, internal/tui/session/model_menu_test.go.
package session

import (
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/tui/locale"
	"github.com/rivo/uniseg"
)

func TestLocalizedThinkingChoicePositions(t *testing.T) {
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		a := fixture(t)
		a.language = lang
		on := true
		a.selection.Thinking = &on
		a.selection.ThinkingSupported = true
		a.openModel()
		a.render()
		label := lang.Text("Thinking") + " " + lang.Text("ON") + " / " + lang.Text("OFF")
		if shown := screenLine(a.screen, 3, 80); !strings.Contains(shown, label) {
			t.Fatal(shown)
		}
		x := uniseg.StringWidth(lang.Text("Thinking") + " ")
		_, selected, _ := a.screen.Get(x, 3)
		x += uniseg.StringWidth(lang.Text("ON") + " / ")
		_, other, _ := a.screen.Get(x, 3)
		if !selected.HasBold() || !other.HasDim() {
			t.Fatal("translated selection emphasis lost")
		}
	}
}
