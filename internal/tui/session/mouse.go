// Purpose: Route pointer scrolling after scrollbar and text-selection hit testing.
// Pattern: Focused terminal input adapter.
// Usage: The event loop sends mouse events here; keyboard focus uses focusKind.
// Related: internal/tui/session/keys.go, internal/tui/session/scrollbar.go, internal/tui/session/text_selection.go.
package session

import (
	"github.com/gdamore/tcell/v3"
)

func (a *app) mouse(e *tcell.EventMouse) {
	if a.scrollbarMouse(e) {
		return
	}
	if a.selectionMouse(e) {
		return
	}
	n := 0
	if e.Buttons()&tcell.WheelUp != 0 {
		n = -3
	}
	if e.Buttons()&tcell.WheelDown != 0 {
		n = 3
	}
	if n == 0 {
		return
	}
	if a.page == "files" && a.files.filePreview != nil {
		a.files.filePreview.Offset = max(0, a.files.filePreview.Offset+n)
		return
	}
	if a.page == "files" {
		a.files.browser.Move(n)
		return
	}
	if a.page != "chat" {
		return
	}
	w, h := a.screen.Size()
	a.scroll(n, w, a.chatHeight(w, a.bodyHeight(h)))
}
