// Purpose: Project viewport and streaming values while reserving composer and candidate rows.
// Pattern: Presentation renderer.
// Usage: Called by render; scrolling uses the same reserved transcript height.
// Related: internal/tui/session/render.go, internal/tui/session/render_hints.go, internal/tui/view/input.go, internal/tui/viewport/viewport.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/tui/view"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

func (a *app) chatHeight(width, height int) int {
	window := a.editor.Window(width, min(a.opts.Config.UI.InputMaxLines, max(1, height-1)))
	return view.Composer(len(window.Rows), height, a.candidates(), a.stopNotice).TranscriptRows
}

func (a *app) renderChat(width, height int) {
	lines := a.chatLines(view.TranscriptWidth(width))
	window := a.editor.Window(width, min(a.opts.Config.UI.InputMaxLines, max(1, height-1)))
	layout := view.Composer(len(window.Rows), height, a.candidates(), a.stopNotice)
	a.inputRegion = inputRegion{editor: &a.editor, window: window, y: layout.InputY, width: width, revision: a.editor.Revision()}
	available := layout.TranscriptRows
	start := a.output.Start(lines, available)
	if a.scrollbar.x != width-1 {
		a.scrollbar.dragging = false
	}
	a.scrollbar.x, a.scrollbar.y = width-1, 1
	a.scrollbar.bar = viewport.NewScrollbar(len(lines), available, start)
	if !a.scrollbar.bar.Visible() {
		a.scrollbar.dragging = false
	}
	if a.selectedTranscript != nil && (a.textSelection.Active() || a.textSelection.Dragging()) {
		lines, start = a.selectedTranscript, 0
	}
	view.Chat(a.screen, view.ChatDisplay{Notice: a.stopNotice, Activity: a.activity(), Usage: a.displayTokens(), Lines: lines, Start: start, Scrollbar: a.scrollbar.bar, Input: window, Layout: layout, Candidates: a.candidates()}, width)
	a.captureChatSources(lines, start, available)
}
