// Purpose: Project viewport and streaming values while reserving composer and candidate rows.
// Pattern: Presentation renderer.
// Usage: Called by render; scrolling uses the same reserved transcript height.
// Related: internal/tui/session/render.go, internal/tui/session/render_hints.go, internal/tui/view/input.go, internal/tui/viewport/viewport.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/input"
	"github.com/keonho-kim/meca/internal/tui/view"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (a *app) chatHeight(width, height int) int {
	_, layout, _ := a.chatComposer(width, height)
	return layout.TranscriptRows
}

func (a *app) chatComposer(width, height int) (input.Window, view.ComposerLayout, []string) {
	window := a.editor.Window(width, min(a.opts.Config.UI.InputMaxLines, max(1, height-1)))
	references := a.selectedReferenceLines(width, height-len(window.Rows)-2)
	return window, view.Composer(len(window.Rows), height, a.candidates(), a.stopNotice, len(references)), references
}

func (a *app) renderChat(width, height int) {
	transcript := a.chatLayout(view.TranscriptWidth(width))
	window, layout, references := a.chatComposer(width, height)
	a.inputRegion = inputRegion{editor: &a.editor, window: window, y: layout.InputY, width: width, revision: a.editor.Revision()}
	available := layout.TranscriptRows
	start := transcript.Start(&a.output, available)
	lines := transcript.Window(start, available)
	if a.scrollbar.x != width-1 {
		a.scrollbar.dragging = false
	}
	a.scrollbar.x, a.scrollbar.y = width-1, 1
	a.scrollbar.bar = viewport.NewScrollbar(transcript.Count(), available, start)
	if !a.scrollbar.bar.Visible() {
		a.scrollbar.dragging = false
	}
	if a.selectedTranscript != nil && (a.textSelection.Active() || a.textSelection.Dragging()) {
		lines = a.selectedTranscript
	}
	view.Chat(a.screen, view.ChatDisplay{Notice: a.stopNotice, Activity: a.activity(), Usage: a.displayTokens(), Lines: lines, Scrollbar: a.scrollbar.bar, Input: window, References: references, Layout: layout, Candidates: a.candidates()}, width)
	a.captureChatSources(lines, available)
}
