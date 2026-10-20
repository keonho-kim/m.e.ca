// Purpose: Reuse transcript and active-response layouts while only input or status cursors change.
// Pattern: Cached presentation projection.
// Usage: renderChat requests lines; worker events continue to own source values.
// Related: internal/tui/session/render_chat.go, internal/tui/viewport/viewport.go.
package session

import "github.com/keonho-kim/v-code/internal/tui/viewport"

type chatProjection struct {
	base                     *viewport.Line
	count, width, next       int
	thinking, stream, status viewport.Block
	pending                  string
	lines                    []viewport.Line
}

func (a *app) chatLines(width int) []viewport.Line {
	base := a.output.Lines(width)
	var first *viewport.Line
	if len(base) > 0 {
		first = &base[0]
	}
	p := &a.chatProjection
	if p.width == width && p.base == first && p.count == len(base) && p.next == a.output.Next && p.thinking.Text == a.thinking && p.stream.Text == a.stream && p.status.Text == a.status && p.pending == a.pending {
		return p.lines
	}
	p.width, p.base, p.count = width, first, len(base)
	p.next = a.output.Next
	p.thinking.Text, p.thinking.Thinking = a.thinking, true
	p.stream.Text, p.stream.Markdown = a.stream, true
	p.status.Text = a.status
	p.pending = a.pending
	lines := base[:len(base):len(base)]
	next := a.output.Next + 1
	for _, block := range []*viewport.Block{&p.thinking, &p.stream, &p.status} {
		block.ID = next
		projected := block.Lines(width)
		if len(projected) > 0 {
			lines = viewport.AppendLines(lines, projected)
			next++
		}
	}
	if a.pending != "" {
		block := viewport.Block{ID: next, Text: a.language.Text("예약 > ") + a.opts.Vault.Redact(a.pending), Thinking: true}
		lines = viewport.AppendLines(lines, block.Lines(width))
	}
	p.lines = lines
	return lines
}
