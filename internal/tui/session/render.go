// Purpose: Coordinate session screens and bounded output projections from event-loop state.
// Pattern: Presentation renderer.
// Usage: Called by the terminal event loop after input or worker events.
// Related: internal/tui/session/app.go, internal/tui/viewport/viewport.go, internal/tui/session/render_files.go, internal/tui/session/render_chat.go, internal/tui/session/keymap.go, internal/tui/session/render_hints.go, internal/tui/session/render_menu.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/view"
)

func (a *app) render() {
	if a.selectionPage != a.page {
		a.textSelection.Clear()
	}
	a.screen.Clear()
	a.screen.HideCursor()
	w, h := a.screen.Size()
	if w < 12 || h < 4 {
		a.textSelection.Clear()
		view.Text(a.screen, 0, 0, w, "Resize terminal", tcell.StyleDefault)
		a.screen.Show()
		return
	}
	body := a.bodyHeight(h)
	switch a.page {
	case "files":
		a.renderFiles(w, body)
	case "chat":
		a.renderChat(w, body)
	default:
		a.renderMenu(w, body)
	}
	a.textSelection.Draw(a.screen)
	view.Footer(a.screen, view.Status{Model: a.selection.Model, Thinking: a.selection.Thinking != nil && *a.selection.Thinking, Effort: a.selection.Effort(), Settings: a.settings}, a.shortcuts(), body, w)
	a.screen.Show()
}
