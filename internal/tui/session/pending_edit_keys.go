// Purpose: Route queued-message editing keys before chat commands or active-agent controls.
// Pattern: Keyboard input adapter.
// Usage: Chat key routing calls pendingEditKey after clipboard and process-exit handling.
// Related: internal/tui/session/keys.go, internal/tui/session/pending_edit.go, internal/tui/input/keys.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/input"
)

func (a *app) pendingEditKey(e *tcell.EventKey) bool {
	if a.page != "chat" {
		return false
	}
	if e.Modifiers() == tcell.ModShift && (e.Key() == tcell.KeyLeft || e.Key() == tcell.KeyRight) {
		if a.pendingEdit == nil && (!a.canEditPending() || e.Key() == tcell.KeyRight) {
			return false
		}
		delta := -1
		if e.Key() == tcell.KeyRight {
			delta = 1
		}
		a.navigatePending(delta)
		return true
	}
	if a.pendingEdit == nil {
		return false
	}
	switch e.Key() {
	case tcell.KeyEscape:
		a.closePendingEdit()
	case tcell.KeyCtrlD:
		a.deletePendingEdit()
	case tcell.KeyEnter:
		if e.Modifiers() == tcell.ModShift {
			a.editor.Insert("\n")
		} else if e.Modifiers() == 0 {
			a.savePendingEdit()
		}
	case tcell.KeyTab:
		if e.Modifiers() == 0 {
			a.savePendingEdit()
		}
	case tcell.KeyUp, tcell.KeyDown:
		if e.Modifiers() == 0 {
			delta := -1
			if e.Key() == tcell.KeyDown {
				delta = 1
			}
			width, _ := a.screen.Size()
			a.editor.MoveVertical(delta, width)
		}
	case tcell.KeyCtrlO, tcell.KeyPgUp, tcell.KeyPgDn:
		return false
	default:
		input.EditKey(&a.editor, e)
	}
	return true
}
