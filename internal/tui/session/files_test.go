// Purpose: Verify file-view quit input works with either focus without submitting the chat draft.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session -run TestFileViewQuit.
// Related: internal/tui/session/files.go, internal/tui/session/render_files.go, internal/tui/session/keys.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"strings"
	"testing"
)

func TestFileViewQuitFromBrowserPreservesDraft(t *testing.T) {
	a := fixture(t)
	a.openFiles(".")
	a.editor.Set("existing draft")
	a.busy = true
	typeInput(a, "ignored")
	typeInput(a, ":q")
	a.render()
	if a.page != "files" || !strings.Contains(screenLine(a.screen, 21, 80), ":q") {
		t.Fatal("quit command was hidden or executed before Enter")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "chat" || a.exit || !a.busy || a.editor.String() != "existing draft" {
		t.Fatal("quit did not preserve the active session and draft")
	}
}
func TestFileViewQuitCanCorrectAnIncorrectCommand(t *testing.T) {
	a := fixture(t)
	a.openFiles(".")
	typeInput(a, ":x")
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", 0))
	typeInput(a, "q")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "chat" {
		t.Fatal("corrected quit was not recognized")
	}
}
func TestFileViewQuitFromInputFocus(t *testing.T) {
	a := fixture(t)
	a.openFiles(".")
	a.keyEvent(tcell.NewEventKey(tcell.KeyBacktab, "", 0))
	typeInput(a, ":q")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "chat" || a.exit {
		t.Fatal("input-focus quit did not return to chat")
	}
}
