// Purpose: Route menu and connection-field keys without submitting incomplete input or bypassing approvals.
// Pattern: Presentation input handler.
// Usage: Called by the session key router when a menu owns focus.
// Related: internal/tui/session/keys.go, internal/tui/session/menus.go, internal/tui/session/model.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/input"
)

func (a *app) menuKey(e *tcell.EventKey) {
	if a.page == "approval" || a.page == "modelinfo" {
		if e.Key() == tcell.KeyPgDn {
			a.previewOffset += 8
			return
		}
		if e.Key() == tcell.KeyPgUp {
			a.previewOffset = max(0, a.previewOffset-8)
			return
		}
	}
	if a.paste {
		if e.Key() == tcell.KeyRune {
			a.menuInput.Insert(e.Str())
		}
		return
	}
	if e.Key() == tcell.KeyEscape {
		if a.page == "approval" {
			a.decide(false)
		} else if a.page == "modelinfo" {
			a.openModel()
		} else {
			a.page = "chat"
			a.key = ""
		}
		a.menuInput.Set("")
		return
	}
	if a.page == "endpoint" || a.page == "key" || a.page == "manual" {
		if e.Key() == tcell.KeyCtrlU {
			a.menuInput.Set("")
			return
		}
		if e.Key() == tcell.KeyEnter {
			if a.menuInput.String() == ":q" {
				a.page = "chat"
				a.key = ""
				a.menuInput.Set("")
				return
			}
			a.confirmField()
			return
		}
		input.EditKey(&a.menuInput, e)
		return
	}
	if a.page == "model" && a.choice == 1 && a.selection.ThinkingSupported && (e.Key() == tcell.KeyLeft || e.Key() == tcell.KeyRight) {
		a.thinkingChoice = e.Key() == tcell.KeyLeft
		return
	}
	switch e.Key() {
	case tcell.KeyUp:
		a.choice = max(0, a.choice-1)
	case tcell.KeyDown:
		a.choice = min(max(0, len(a.choices)-1), a.choice+1)
	case tcell.KeyEnter:
		if a.menuInput.String() == ":q" {
			if a.page == "approval" {
				a.decide(false)
			} else {
				a.page = "chat"
				a.key = ""
			}
			a.menuInput.Set("")
			return
		}
		if len(a.choices) > 0 {
			if a.page == "modelinfo" {
				a.openModel()
			} else {
				a.choose()
			}
		}
	default:
		input.EditKey(&a.menuInput, e)
	}
}
func (a *app) decide(approved bool) {
	select {
	case a.decision <- approved:
	default:
	}
	a.proposal = nil
	a.page = "chat"
	a.key = ""
}
