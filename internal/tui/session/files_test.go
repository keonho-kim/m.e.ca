// Purpose: Verify file navigation closes through Esc and never consumes request text as navigation commands.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session -run TestFileView.
// Related: internal/tui/session/files.go, internal/tui/session/render_files.go, internal/tui/session/keys.go.
package session

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

func TestFileViewEscapePreservesDraft(t *testing.T) {
	a := fixture(t)
	a.openFiles(".")
	a.compose.editor.Set("existing draft")
	a.work.state = workerSetup
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "chat" || a.exit || !a.work.state.active() || a.compose.editor.String() != "existing draft" {
		t.Fatal("closing the browser did not preserve the active session and draft")
	}
}
func TestFileViewIgnoresNavigationCommands(t *testing.T) {
	a := fixture(t)
	a.openFiles(".")
	finishBrowserIO(t, a)
	a.compose.editor.Set("existing draft")
	typeInput(a, ":q")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "files" || a.menuInput.String() != "" || a.compose.editor.String() != "existing draft" {
		t.Fatal("browser focus still accepts a navigation command buffer")
	}
}
func TestFileViewEscapeFromInputFocus(t *testing.T) {
	a := fixture(t)
	a.openFiles(".")
	a.keyEvent(tcell.NewEventKey(tcell.KeyBacktab, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "chat" || a.exit {
		t.Fatal("input-focus Escape did not return to chat")
	}
}
