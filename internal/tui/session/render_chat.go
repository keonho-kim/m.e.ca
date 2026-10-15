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
	rows := view.InputRows(a.editor, max(1, width), min(a.opts.Config.UI.InputMaxLines, max(1, height-2)))
	hints := 0
	if a.commandOpen {
		hints = max(1, len(a.commandHints))
	} else if a.refOpen {
		hints = max(1, len(a.refs))
	}
	hints = min(hints, view.MaxCandidateRows, max(0, height-rows-2))
	// Reserve the fixed token row and the gap before the composer.
	return max(0, height-rows-hints-2)
}

func (a *app) renderChat(width, height int) {
	lines := a.output.Lines(width)
	appendTemporary := func(value string, thinking bool) {
		if value == "" {
			return
		}
		for _, line := range viewport.Wrap(value, width) {
			lines = append(lines, viewport.Line{Text: line, Thinking: thinking})
		}
	}
	appendTemporary(a.thinking, true)
	stream := viewport.Block{Text: a.stream, Markdown: true}
	if a.stream != "" {
		lines = append(lines, stream.Lines(width)...)
	}
	appendTemporary(a.status, false)
	if a.pending != "" {
		appendTemporary("예약 > "+a.opts.Vault.Redact(a.pending), true)
	}
	available := a.chatHeight(width, height)
	start := a.output.Start(lines, available)
	view.Chat(a.screen, view.ChatDisplay{Activity: a.activity(), Usage: a.displayTokens(), Lines: lines, Start: start, Available: available, Editor: a.editor, InputMaxLines: a.opts.Config.UI.InputMaxLines, Candidates: a.candidates()}, width, height)
}
