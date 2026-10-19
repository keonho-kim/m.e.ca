// Purpose: Verify menus use localized labels while keeping settings, identifiers and text literal.
// Pattern: Simulated terminal integration tests.
// Usage: go test ./internal/tui/session -run TestLocalized.
// Related: internal/tui/session/render_menu.go, internal/tui/session/settings.go, internal/tui/locale/locale.go.
package session

import (
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/tui/locale"
)

func TestLocalizedPermissionMenuPreservesValues(t *testing.T) {
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		a := fixture(t)
		a.language = lang
		a.page = "permission"
		a.choices = []string{"approve", "allow-all"}
		a.choice = 1
		a.render()
		var rows []string
		for y := 0; y < 24; y++ {
			rows = append(rows, screenLine(a.screen, y, 80))
		}
		text := strings.Join(rows, "\n")
		if !strings.Contains(text, lang.Text("allow-all")) || strings.Contains(text, "allow-all") {
			t.Fatal(text)
		}
		a.choose()
		if a.changeValue != "allow-all" {
			t.Fatal("display translation changed persisted value")
		}
		a.choose()
		if a.settings.Permission != policy.AllowAll {
			t.Fatal("translated choice changed policy")
		}
	}
}

func TestLocalizedLabelsPreserveLiteralModelAndDraft(t *testing.T) {
	a := fixture(t)
	a.language = locale.Korean
	a.selection.Model = "allow-all"
	a.editor.Set("Thinking ON / OFF allow-all")
	a.render()
	if got := screenLine(a.screen, 22, 80); !strings.Contains(got, "allow-all") {
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
		{locale.Korean, "!승인", "!permission"},
		{locale.English, "!permission", "!permission"},
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
