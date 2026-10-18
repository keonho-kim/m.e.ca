// Purpose: Verify multiline navigation and focused cancellation preserve user work.
// Pattern: Virtual terminal behavior tests.
// Usage: go test ./internal/tui/session -run 'TestHistoryModifiers|TestCandidateEscape|TestReleasedKey'.
// Related: internal/tui/session/keys.go, internal/tui/session/menu_keys.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"testing"
)

func TestHistoryModifiers(t *testing.T) {
	for _, modifier := range []tcell.ModMask{tcell.ModCtrl, tcell.ModMeta} {
		a := fixture(t)
		a.inputs, a.historyIndex = []string{"old"}, 1
		a.editor.Set("abc\ndef")
		a.keyEvent(tcell.NewEventKey(tcell.KeyUp, "", 0))
		if a.editor.String() != "abc\ndef" || a.editor.Cursor != 3 {
			t.Fatal("plain Up did not move vertically")
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyUp, "", modifier))
		if a.editor.String() != "old" {
			t.Fatal("modified Up did not recall history")
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", modifier))
		if a.editor.String() != "abc\ndef" {
			t.Fatal("history lost draft")
		}
	}
}

func TestCandidateEscapeLeavesTaskRunning(t *testing.T) {
	a := fixture(t)
	a.busy, a.agentActive = true, true
	cancelled := false
	a.taskCancel = func() { cancelled = true }
	typeInput(a, "!mo")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if cancelled || a.commandOpen {
		t.Fatal("candidate dismissal stopped work")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if !cancelled {
		t.Fatal("second Escape did not stop work")
	}
}

func TestReleasedKeyDoesNotEdit(t *testing.T) {
	a := fixture(t)
	a.keyEvent(tcell.NewEventKeyEx(tcell.KeyRune, "x", 0, false, 0, 1))
	if a.editor.String() != "" {
		t.Fatal("key release inserted text")
	}
}
