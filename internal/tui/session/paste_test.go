// Purpose: Verify bracketed paste is one literal edit and never dispatches pasted commands.
// Pattern: Terminal input contract tests.
// Usage: go test ./internal/tui/session -run TestPaste.
// Related: internal/tui/session/paste.go, internal/tui/session/event_loop.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"testing"
)

func TestPasteCommitsOneEditWithoutExecuting(t *testing.T) {
	a := fixture(t)
	a.editor.Set("draft ")
	revision := a.editor.Revision()
	a.terminalEvent(tcell.NewEventPaste(true))
	typeInput(a, "!quit")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	typeInput(a, "한글 👩‍💻")
	if a.exit || a.editor.String() != "draft " {
		t.Fatal("paste executed or edited before closing boundary")
	}
	a.terminalEvent(tcell.NewEventPaste(false))
	if a.exit || a.editor.String() != "draft !quit\n한글 👩‍💻" || a.editor.Revision() != revision+1 {
		t.Fatal("paste was not one literal edit")
	}
}

func TestPasteKeepsTerminalLineFeed(t *testing.T) {
	a := fixture(t)
	a.terminalEvent(tcell.NewEventPaste(true))
	typeInput(a, "first")
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlJ, "", 0))
	typeInput(a, "second")
	a.terminalEvent(tcell.NewEventPaste(false))
	if a.editor.String() != "first\nsecond" {
		t.Fatalf("terminal LF lost: %q", a.editor.String())
	}
}
