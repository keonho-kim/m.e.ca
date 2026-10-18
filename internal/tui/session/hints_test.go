// Purpose: Verify command hints above the fixed editor and foreground-only selection across terminal views.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session -run 'TestCommandHints|TestForegroundOnly'.
// Related: internal/tui/session/hints.go, internal/tui/session/render.go, internal/tui/view/styles.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tools/local"
	"strings"
	"testing"
)

func typeInput(a *app, value string) {
	for _, r := range value {
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, string(r), 0))
	}
}
func TestCommandHints(t *testing.T) {
	a := fixture(t)
	typeInput(a, "!")
	a.render()
	if !strings.Contains(screenLine(a.screen, 16, 80), "!model") || !strings.Contains(screenLine(a.screen, 18, 80), "!dir") || !strings.Contains(screenLine(a.screen, 20, 80), "!quit") {
		t.Fatal("five commands missing above editor")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if a.editor.String() != "!model" || a.page != "chat" {
		t.Fatal("completion executed the command")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "model" {
		t.Fatalf("completed command did not execute: %s", a.page)
	}
	a.page = "chat"
	typeInput(a, "!per")
	a.render()
	if !strings.Contains(screenLine(a.screen, 20, 80), "!permission") || strings.Contains(screenLine(a.screen, 4, 80), "!model") {
		t.Fatal("prefix did not filter commands")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.editor.String() != "" || a.page != "permission" {
		t.Fatal("Enter must execute the selected command")
	}
	a.page = "chat"
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, "", 0))
	typeInput(a, "!")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.render()
	if a.editor.String() != "!" || strings.Contains(screenLine(a.screen, 16, 80), "!model") {
		t.Fatal("Escape must preserve input and dismiss hints")
	}
}
func TestCommandHintsIgnoreTextArgumentsAndPaste(t *testing.T) {
	for _, value := range []string{"explain !model", "!dir src/", "!model\ntext"} {
		a := fixture(t)
		typeInput(a, value)
		a.render()
		if strings.Contains(screenLine(a.screen, 16, 80), "!model") {
			t.Fatalf("hints opened in message or argument: %q", value)
		}
	}
	a := fixture(t)
	a.pasteBoundary(true)
	typeInput(a, "!")
	a.render()
	if strings.Contains(screenLine(a.screen, 16, 80), "!model") {
		t.Fatal("paste opened interactive hints")
	}
}
func TestCommandHintsPreserveShiftEnter(t *testing.T) {
	a := fixture(t)
	typeInput(a, "!")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModShift))
	a.render()
	if a.editor.String() != "!\n" || strings.Contains(screenLine(a.screen, 4, 80), "!model") {
		t.Fatal("Shift+Enter completed a command instead of inserting a newline")
	}
}
func TestForegroundOnlyViews(t *testing.T) {
	a := fixture(t)
	a.output.Append("thinking", true)
	a.render()
	assertForegroundOnly(t, a.screen)
	a.page, a.choices = "model", []string{"first", "second"}
	a.render()
	assertForegroundOnly(t, a.screen)
	_, style, _ := a.screen.Get(0, 2)
	attrs := style.GetAttributes()
	if attrs&tcell.AttrBold == 0 {
		t.Fatal("selected menu row lacks foreground emphasis")
	}
	a.page = "chat"
	a.page = "files"
	a.browser.Open(a.opts.Session.ProjectRoot, []local.Entry{{Name: "file.go"}}, false)
	a.render()
	assertForegroundOnly(t, a.screen)
}
func assertForegroundOnly(t *testing.T, screen tcell.Screen) {
	t.Helper()
	w, h := screen.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			_, style, _ := screen.Get(x, y)
			background, attrs := style.GetBackground(), style.GetAttributes()
			if background != tcell.ColorDefault || attrs&tcell.AttrReverse != 0 {
				t.Fatalf("background/reverse at %d,%d: %v", x, y, style)
			}
		}
	}
}
