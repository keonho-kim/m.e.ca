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
	return view.Composer(a.editor, width, height, a.opts.Config.UI.InputMaxLines, a.candidates(), a.stopNotice).TranscriptRows
}

func (a *app) renderChat(width, height int) {
	lines := a.output.Lines(width)
	appendTemporary := func(value string, thinking bool) {
		block := viewport.Block{Text: value, Thinking: thinking}
		lines = viewport.AppendLines(lines, block.Lines(width))
	}
	appendTemporary(a.thinking, true)
	stream := viewport.Block{Text: a.stream, Markdown: true}
	if a.stream != "" {
		lines = viewport.AppendLines(lines, stream.Lines(width))
	}
	appendTemporary(a.status, false)
	if a.pending != "" {
		appendTemporary("예약 > "+a.opts.Vault.Redact(a.pending), true)
	}
	available := a.chatHeight(width, height)
	start := a.output.Start(lines, available)
	view.Chat(a.screen, view.ChatDisplay{Notice: a.stopNotice, Activity: a.activity(), Usage: a.displayTokens(), Lines: lines, Start: start, Available: available, Editor: a.editor, InputMaxLines: a.opts.Config.UI.InputMaxLines, Candidates: a.candidates()}, width, height)
}
