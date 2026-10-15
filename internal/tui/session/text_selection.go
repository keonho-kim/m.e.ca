// Purpose: Route visible text dragging and clipboard requests ahead of application shortcuts.
// Pattern: Session presentation controller.
// Usage: Mouse and keyboard handlers call these methods on the event-loop owner.
// Related: internal/tui/selection/selection.go, internal/tui/session/keys.go, internal/tui/session/keymap.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/input"
	"strings"
)

func (a *app) selectionMouse(e *tcell.EventMouse) bool {
	if a.page != "chat" && a.page != "files" {
		a.textSelection.Clear()
		return false
	}
	if e.Buttons()&(tcell.WheelUp|tcell.WheelDown) != 0 {
		a.textSelection.Clear()
		return false
	}
	x, y := e.Position()
	if e.Buttons()&tcell.Button1 != 0 {
		if a.textSelection.Dragging() {
			a.textSelection.Move(x, y)
		} else {
			a.selectionPage = a.page
			_, h := a.screen.Size()
			a.textSelection.Begin(a.screen, x, y, 1, a.bodyHeight(h)-1)
		}
		return true
	}
	if a.textSelection.Dragging() {
		a.textSelection.Move(x, y)
		a.textSelection.Release()
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
			a.textSelection.Clear()
			return true
		}
	}
	// Editing, navigation or a view switch invalidates screen coordinates.
	a.textSelection.Clear()
	return false
}
