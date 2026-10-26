// Purpose: Route visible text dragging and clipboard requests ahead of application shortcuts.
// Pattern: Session presentation controller.
// Usage: Mouse and keyboard handlers call these methods on the event-loop owner.
// Related: internal/tui/selection/selection.go, internal/tui/session/keys.go, internal/tui/session/keymap.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/input"
	"slices"
	"strings"
)

func (a *app) selectionMouse(e *tcell.EventMouse) bool {
	if a.page != "chat" && a.page != "files" && a.page != "model" {
		a.clearSelection()
		return false
	}
	if e.Buttons()&(tcell.WheelUp|tcell.WheelDown) != 0 {
		a.clearSelection()
		return false
	}
	x, y := e.Position()
	if e.Buttons()&tcell.Button1 != 0 {
		if a.textSelection.Dragging() {
			a.textSelection.Move(x, y)
		} else {
			_, h := a.screen.Size()
			body := a.bodyHeight(h)
			if a.footerRows > 0 {
				body = max(0, h-a.footerRows)
			}
			a.clearSelection()
			a.selectionPage = a.page
			a.textSelection.Begin(a.screen, x, y, 1, body-1)
			a.textSelection.Sources(a.selectionSources)
			if a.inputRegion.contains(x, y) {
				a.selectionInput = a.inputRegion
			} else if a.page == "chat" {
				a.selectedTranscript = slices.Clone(a.chatVisible)
			}
		}
		return true
	}
	if a.textSelection.Dragging() {
		a.textSelection.Move(x, y)
		a.textSelection.Release()
		if !a.textSelection.Active() && a.selectionInput.contains(x, y) && a.selectionInput.editor.Revision() == a.selectionInput.revision {
			r := a.selectionInput
			r.editor.MoveTo(r.window.Offset(x-r.x, y-r.y, false))
		}
		return true
	}
	return false
}
func (a *app) selectionKey(e *tcell.EventKey) bool {
	if !e.Pressed() {
		return false
	}
	if a.textSelection.Active() {
		// Meta+C is accepted wherever a terminal delivers it; on macOS this is Cmd+C.
		copyKey := input.ExitKey(e) || e.Key() == tcell.KeyRune && strings.EqualFold(e.Str(), "c") && e.Modifiers() == tcell.ModMeta
		if copyKey {
			text := a.textSelection.Text()
			if text != "" {
				a.copySelection(text)
			}
			return true
		}
		if e.Key() == tcell.KeyEscape {
			a.clearSelection()
			a.stopNotice = ""
			return true
		}
		if e.Key() == tcell.KeyBackspace || e.Key() == tcell.KeyDelete {
			a.deleteSelection()
			a.clearSelection()
			a.stopNotice = ""
			return true
		}
	}
	// Editing, navigation or a view switch invalidates screen coordinates.
	a.clearSelection()
	return false
}
