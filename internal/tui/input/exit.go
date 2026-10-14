// Purpose: Recognize app-exit keystrokes actually delivered by a terminal.
// Pattern: Pure input classification.
// Usage: Shared by the active session and startup resume screen.
// Related: internal/tui/session/keys.go, internal/tui/startup/resume.go.
package input

import (
	"github.com/gdamore/tcell/v3"
	"runtime"
	"strings"
)

func ExitKey(event *tcell.EventKey) bool {
	if !event.Pressed() {
		return false
	}
	modifiers := event.Modifiers() & (tcell.ModCtrl | tcell.ModAlt | tcell.ModShift | tcell.ModMeta | tcell.ModHyper)
	if event.Key() == tcell.KeyCtrlC {
		return modifiers == 0 || modifiers == tcell.ModCtrl
	}
	if event.Key() != tcell.KeyRune || !strings.EqualFold(event.Str(), "c") {
		return false
	}
	return modifiers == tcell.ModCtrl || runtime.GOOS == "darwin" && modifiers == tcell.ModMeta
}
