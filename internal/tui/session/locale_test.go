// Purpose: Verify menus use localized labels while keeping settings, identifiers and text literal.
// Pattern: Simulated terminal integration tests.
// Usage: go test ./internal/tui/session -run TestLocalized.
// Related: internal/tui/session/render_menu.go, internal/tui/session/settings.go, internal/tui/locale/locale.go.
package session

import (
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/tui/locale"
)

func TestLocalizedStatusOmitsFixedApprovalPolicy(t *testing.T) {
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		a := fixture(t)
		a.language = lang
		a.selection.Model = "chosen-model"
		a.render()
		text := screenLine(a.screen, 24-a.footerRows, 80)
		if strings.TrimSpace(text) != "chosen-model" {
			t.Fatal(text)
		}
	}
}

func TestLocalizedLabelsPreserveLiteralModelAndDraft(t *testing.T) {
	a := fixture(t)
	a.language = locale.Korean
	a.selection.Model = "allow-all"
	a.editor.Set("Thinking ON / OFF allow-all")
	a.render()
	if got := screenLine(a.screen, 24-a.footerRows, 80); !strings.Contains(got, "allow-all") {
		t.Fatal(got)
	}
	if a.editor.String() != "Thinking ON / OFF allow-all" {
		t.Fatal("draft translated")
	}
	if got := screenLine(a.screen, 0, 80); !strings.Contains(got, "상태 대기") || !strings.Contains(got, "입력 0 · 출력 0") {
		t.Fatal(got)
	}
}

func TestLocalizedCommandSearch(t *testing.T) {
	for _, test := range []struct {
		lang        locale.Language
		query, want string
	}{
		{locale.Korean, "!추론", "!model"},
		{locale.English, "!thinking", "!model"},
		{locale.Korean, "!파일", "!dir"},
		{locale.English, "!browse", "!dir"},
	} {
		a := fixture(t)
		a.language = test.lang
		a.editor.Set(test.query)
		a.refreshHints()
		if len(a.commandHints) != 1 || a.commandHints[0].Name != test.want {
			t.Fatalf("%q: %+v", test.query, a.commandHints)
		}
	}
}
