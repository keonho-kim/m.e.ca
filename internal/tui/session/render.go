// Purpose: Coordinate session screens and bounded output projections from event-loop state.
// Pattern: Presentation renderer.
// Usage: Called by the terminal event loop after input or worker events.
// Related: internal/tui/session/app.go, internal/tui/viewport/viewport.go, internal/tui/session/render_files.go, internal/tui/session/render_chat.go, internal/tui/session/keymap.go, internal/tui/session/render_hints.go, internal/tui/session/render_menu.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/textlayout"
	"github.com/keonho-kim/v-code/internal/tui/view"
)

func (a *app) render() {
	if a.selectionPage != a.page {
		a.clearSelection()
	}
	a.inputRegion = inputRegion{}
	a.selectionSources = make(map[int]textlayout.Fragment)
	a.screen.Clear()
	a.screen.HideCursor()
	w, h := a.screen.Size()
	if w < 12 || h < 4 {
		a.clearSelection()
		view.Text(a.screen, 0, 0, w, "Resize terminal", tcell.StyleDefault)
		a.screen.Show()
		return
	}
	body := a.bodyHeight(h)
	if a.helpOpen {
		view.KeyboardHelp(a.screen, a.shortcuts(), w, body, a.helpOffset)
		view.Footer(a.screen, view.Status{Model: a.selection.Model, Settings: a.settings}, []view.Shortcut{{Key: "^C", Label: "종료"}, {Key: "Esc", Label: "닫기"}, {Key: "F1", Label: "닫기"}, {Key: "↑↓", Label: "스크롤"}}, body, w)
		a.screen.Show()
		return
	}
	switch a.page {
	case "files":
		a.renderFiles(w, body)
	case "chat":
		a.renderChat(w, body)
	default:
		a.renderMenu(w, body)
	}
	a.captureInputSources()
	a.textSelection.Draw(a.screen)
	view.Footer(a.screen, view.Status{Model: a.selection.Model, Thinking: a.selection.Thinking != nil && *a.selection.Thinking, Effort: a.selection.Effort(), Settings: a.settings}, a.shortcuts(), body, w)
	a.screen.Show()
}
