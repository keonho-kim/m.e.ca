// Purpose: Verify command hints below the editor and foreground-only selection across terminal views.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session -run 'TestCommandHints|TestForegroundOnly'.
// Related: internal/tui/session/hints.go, internal/tui/session/render.go, internal/tui/view/styles.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/keonho-kim/v-code/internal/tools/local"
)

func typeInput(a *app, value string) {
	for _, r := range value {
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, r, 0))
	}
}
func TestCommandHints(t *testing.T) {
	a := fixture(t)
	typeInput(a, "!")
	a.render()
	if !strings.Contains(screenLine(a.screen, 2, 80), "!plan") || !strings.Contains(screenLine(a.screen, 6, 80), "!dir") {
		t.Fatal("five commands missing below editor")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyTAB, 0, 0))
	if a.editor.String() != "!model" || a.page != "chat" {
		t.Fatal("completion executed the command")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if a.page != "model" {
		t.Fatalf("completed command did not execute: %s", a.page)
	}
	a.page = "chat"
	typeInput(a, "!sc")
	a.render()
	if !strings.Contains(screenLine(a.screen, 2, 80), "!scope") || strings.Contains(screenLine(a.screen, 3, 80), "!plan") {
		t.Fatal("prefix did not filter commands")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if a.editor.String() != "!scope" || a.page != "chat" {
		t.Fatal("Enter must insert without executing")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, 0, 0))
	typeInput(a, "!")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	a.render()
	if a.editor.String() != "!" || strings.Contains(screenLine(a.screen, 2, 80), "!plan") {
		t.Fatal("Escape must preserve input and dismiss hints")
	}
}
func TestCommandHintsIgnoreTextArgumentsAndPaste(t *testing.T) {
	for _, value := range []string{"explain !model", "!dir src/", "!model\ntext"} {
		a := fixture(t)
		typeInput(a, value)
		a.render()
		if strings.Contains(screenLine(a.screen, 2, 80), "!plan") {
			t.Fatalf("hints opened in message or argument: %q", value)
		}
	}
	a := fixture(t)
	a.paste = true
	typeInput(a, "!")
	a.render()
	if strings.Contains(screenLine(a.screen, 2, 80), "!plan") {
		t.Fatal("paste opened interactive hints")
	}
}
func TestCommandHintsPreserveAltEnter(t *testing.T) {
	a := fixture(t)
	typeInput(a, "!")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModAlt))
	a.render()
	if a.editor.String() != "!\n" || strings.Contains(screenLine(a.screen, 3, 80), "!plan") {
		t.Fatal("Alt+Enter completed a command instead of inserting a newline")
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
	_, _, style, _ := a.screen.GetContent(0, 2)
	_, _, attrs := style.Decompose()
	if attrs&tcell.AttrBold == 0 {
		t.Fatal("selected menu row lacks foreground emphasis")
	}
	a.page = "chat"
	a.refOpen, a.refs = true, []local.Reference{{Path: "src/"}}
	a.render()
	assertForegroundOnly(t, a.screen)
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
			_, _, style, _ := screen.GetContent(x, y)
			_, background, attrs := style.Decompose()
			if background != tcell.ColorDefault || attrs&tcell.AttrReverse != 0 {
				t.Fatalf("background/reverse at %d,%d: %v", x, y, style)
			}
		}
	}
}
