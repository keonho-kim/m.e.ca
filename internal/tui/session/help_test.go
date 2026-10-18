// Purpose: Verify hidden footer actions remain discoverable and help never cancels the active run.
// Pattern: Virtual terminal interaction tests.
// Usage: go test ./internal/tui/session -run TestKeyboardHelp.
// Related: internal/tui/session/keymap.go, internal/tui/view/help.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"strings"
	"testing"
)

func TestKeyboardHelpAndHistoryHints(t *testing.T) {
	a := fixture(t)
	a.render()
	if !strings.Contains(screenLine(a.screen, 23, 80), "F1") {
		t.Fatal("hidden hints have no help entry")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyF1, "", 0))
	a.render()
	var text strings.Builder
	for y := 0; y < 24; y++ {
		text.WriteString(screenLine(a.screen, y, 80))
	}
	if !strings.Contains(text.String(), "Ctrl/Cmd+↑↓") || !strings.Contains(text.String(), "줄 이동") {
		t.Fatal("history and navigation help missing")
	}
	a.busy, a.agentActive = true, true
	cancelled := false
	a.taskCancel = func() { cancelled = true }
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.helpOpen || cancelled {
		t.Fatal("help dismissal interrupted task")
	}
}
