// Purpose: Apply editing, input recall and request admission keys to the active composer.
// Pattern: Focused keyboard handler.
// Usage: keyEvent dispatches chat and file-composer keys after global and modal handlers.
// Related: internal/tui/session/keys.go, internal/tui/session/focus.go, internal/tui/session/pending_edit_keys.go.
package session

import (
	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/tui/input"
)

func (a *app) composerKey(e *tcell.EventKey, focus focusKind) {
	if (e.Key() == tcell.KeyUp || e.Key() == tcell.KeyDown) && (e.Modifiers() == tcell.ModCtrl || e.Modifiers() == tcell.ModMeta) {
		delta := -1
		if e.Key() == tcell.KeyDown {
			delta = 1
		}
		a.compose.recall(delta)
		a.refreshHints()
		return
	}
	if e.Key() == tcell.KeyTab && e.Modifiers()&tcell.ModShift == 0 && !a.compose.commandOpen && a.work.state.agent() && a.work.state.active() && !a.paste {
		if focus == focusFileInput {
			a.confirmFileInput(true)
		} else {
			a.reserveInput()
		}
		return
	}
	if focus == focusFileInput && (e.Key() == tcell.KeyBacktab || e.Key() == tcell.KeyTab && e.Modifiers()&tcell.ModShift != 0) {
		a.files.browser.Input = false
		return
	}
	if a.hintKey(e) {
		return
	}
	switch e.Key() {
	case tcell.KeyEnter:
		if a.paste || e.Modifiers()&tcell.ModShift != 0 {
			a.compose.editor.Insert("\n")
			a.refreshHints()
		} else {
			if focus == focusFileInput {
				a.confirmFileInput(false)
			} else {
				a.submit()
			}
		}
		return
	case tcell.KeyUp:
		w, _ := a.screen.Size()
		a.compose.editor.MoveVertical(-1, w)
		return
	case tcell.KeyDown:
		w, _ := a.screen.Size()
		a.compose.editor.MoveVertical(1, w)
		return
	case tcell.KeyPgUp:
		w, h := a.screen.Size()
		a.scroll(-max(1, a.chatHeight(w, a.bodyHeight(h))), w, a.chatHeight(w, a.bodyHeight(h)))
		return
	case tcell.KeyPgDn:
		w, h := a.screen.Size()
		a.scroll(max(1, a.chatHeight(w, a.bodyHeight(h))), w, a.chatHeight(w, a.bodyHeight(h)))
		return
	}
	input.EditKey(&a.compose.editor, e)
	a.refreshHints()
}
