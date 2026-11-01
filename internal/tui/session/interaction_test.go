// Purpose: Verify multiline navigation and focused cancellation preserve user work.
// Pattern: Virtual terminal behavior tests.
// Usage: go test ./internal/tui/session -run 'TestHistoryModifiers|TestCandidateEscape|TestReleasedKey'.
// Related: internal/tui/session/keys.go, internal/tui/session/menu_keys.go.
package session

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

func TestHistoryModifiers(t *testing.T) {
	for _, modifier := range []tcell.ModMask{tcell.ModCtrl, tcell.ModMeta} {
		a := screenFixture(t)
		a.compose.inputs, a.compose.historyIndex = []string{"old"}, 1
		a.compose.editor.Set("abc\ndef")
		a.keyEvent(tcell.NewEventKey(tcell.KeyUp, "", 0))
		if a.compose.editor.String() != "abc\ndef" || a.compose.editor.Cursor != 3 {
			t.Fatal("plain Up did not move vertically")
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyUp, "", modifier))
		if a.compose.editor.String() != "old" {
			t.Fatal("modified Up did not recall history")
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", modifier))
		if a.compose.editor.String() != "abc\ndef" {
			t.Fatal("history lost draft")
		}
	}
}

func TestCandidateEscapeLeavesTaskRunning(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerAgent
	cancelled := false
	a.work.cancel = func() { cancelled = true }
	typeInput(a, "!mo")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if cancelled || a.compose.commandOpen {
		t.Fatal("candidate dismissal stopped work")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if !cancelled {
		t.Fatal("second Escape did not stop work")
	}
}

func TestReleasedKeyDoesNotEdit(t *testing.T) {
	a := screenFixture(t)
	a.keyEvent(tcell.NewEventKeyEx(tcell.KeyRune, "x", 0, false, 0, 1))
	if a.compose.editor.String() != "" {
		t.Fatal("key release inserted text")
	}
}
