// Purpose: Coordinate session screens and bounded output projections from event-loop state.
// Pattern: Presentation renderer.
// Usage: Called by the terminal event loop after input or worker events.
// Related: internal/tui/session/app.go, internal/tui/viewport/viewport.go, internal/tui/session/render_files.go, internal/tui/session/render_chat.go, internal/tui/session/keymap.go, internal/tui/session/render_hints.go, internal/tui/session/render_menu.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"github.com/keonho-kim/meca/internal/tui/view"
)

func (a *app) render() {
	if a.page != "chat" {
		a.chatVisible = nil
		if a.chatProjection.layout.Count() > 0 {
			a.output.ReleaseDisplay()
		}
		a.chatProjection = chatProjection{}
	}
	if a.page == "model" && a.modelForm == nil {
		a.initializeModelForm()
	}
	if a.selectionPage != a.page {
		a.clearSelection()
	}
	a.inputRegion = inputRegion{}
	a.selectionSources = make(map[int]textlayout.Fragment)
	a.screen.Clear()
	a.screen.HideCursor()
	w, h := a.screen.Size()
	if a.page != "chat" || w < 12 || h < 4 {
		a.scrollbar = scrollbarState{}
	}
	if w < 12 || h < 4 {
		a.footerRows = 0
		a.clearSelection()
		view.Text(a.screen, 0, 0, w, a.language.Text("Resize terminal"), tcell.StyleDefault)
		a.screen.Show()
		return
	}
	body, shortcuts := a.footerLayout(w, h)
	a.footerRows = h - body
	switch a.page {
	case "files":
		a.renderFiles(w, body)
	case "chat":
		a.renderChat(w, body)
	case "model":
		a.renderModel(w, body)
	case "language":
		a.renderLanguage(w, body)
	case "permission":
		a.renderPermission(w, body)
	default:
		a.renderMenu(w, body)
	}
	a.captureInputSources()
	a.reanchorInputSelection(body)
	a.textSelection.Draw(a.screen)
	view.Footer(a.screen, view.Status{Language: a.language, Model: a.selection.Model}, shortcuts, body, w)
	a.screen.Show()
}
