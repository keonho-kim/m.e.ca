// Purpose: Verify contextual bottom keymaps reserve space and describe actual focused controls.
// Pattern: Terminal layout tests.
// Usage: go test ./internal/tui/session -run TestKeymap.
// Related: internal/tui/session/render.go, internal/tui/session/hints.go, internal/tui/session/render_files.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestKeymapMatchesFocusedScreen(t *testing.T) {
	for _, test := range []struct{ page, want string }{
		{"chat", "전송"}, {"model", "다음 값"}, {"language", "다음 값"}, {"approval", "PgUp/Dn"}, {"files", "입력"},
	} {
		t.Run(test.page, func(t *testing.T) {
			a := fixture(t)
			a.page = test.page
			a.choices = []string{"one", "two"}
			a.proposal = &toolapi.Approval{Name: "Editing", Target: "file", Preview: "change"}
			if test.page == "files" {
				a.openFiles(".")
				finishBrowserIO(t, a)
			}
			a.render()
			footer := footerKeyText(a)
			if !strings.Contains(footer, test.want) || !strings.Contains(footer, "^C") {
				t.Fatalf("footer=%q", footer)
			}
			assertForegroundOnly(t, a.screen)
		})
	}
	a := fixture(t)
	typeInput(a, "!mo")
	a.render()
	footer := footerKeyText(a)
	if !strings.Contains(footer, "실행") || !strings.Contains(footer, "Tab") {
		t.Fatalf("command footer=%q", footer)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "model" || a.editor.String() != "" {
		t.Fatal("Enter completed instead of executing")
	}
}

func TestKeymapKeepsMultilineEditorAboveFooter(t *testing.T) {
	a := fixture(t)
	a.output.Append(strings.Repeat("history\n", 40)+"latest", false)
	a.editor.Set("first\nsecond")
	a.screen.SetSize(32, 12)
	a.render()
	if strings.TrimSpace(screenLine(a.screen, a.inputRegion.y+len(a.inputRegion.window.Rows)-1, 32)) != "second" {
		t.Fatal("editor not above reserved footer")
	}
	if !strings.Contains(footerKeyText(a), "^C") {
		t.Fatal("footer not anchored at bottom")
	}
	a.screen.SetSize(12, 4)
	a.render()
	if a.editor.String() != "first\nsecond" {
		t.Fatal("small terminal changed editor")
	}
}

func footerKeyText(a *app) string {
	w, h := a.screen.Size()
	var text strings.Builder
	for y := h - a.footerRows + 1; y < h; y++ {
		text.WriteString(screenLine(a.screen, y, w))
		text.WriteByte('\n')
	}
	return text.String()
}

func TestCommandCandidateEnterExecutesEveryCommand(t *testing.T) {
	for _, test := range []struct{ name, page string }{
		{"!model", "model"}, {"!language", "language"}, {"!dir", "files"}, {"!quit", "chat"}, {"!new", "chat"},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := fixture(t)
			typeInput(a, "!")
			selected := false
			for index, hint := range a.commandHints {
				if hint.Name == test.name {
					a.commandIndex = index
					selected = true
				}
			}
			if !selected {
				t.Fatal("command missing")
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
			if a.page != test.page || a.commandOpen {
				t.Fatalf("command not executed: page=%s", a.page)
			}
			if (test.name == "!quit" || test.name == "!new") && !a.exit {
				t.Fatal("quit did not exit")
			}
		})
	}
	a := fixture(t)
	a.busy = true
	typeInput(a, "!mo")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "chat" || !a.busy || a.editor.String() != "!model" {
		t.Fatal("command selection bypassed active-task guard")
	}
}
