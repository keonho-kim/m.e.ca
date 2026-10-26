// Purpose: Compose history and live blocks into one sparse, event-loop-owned transcript index.
// Pattern: Presentation projection.
// Usage: Chat rendering, scrolling and text selection share chatLayout.
// Related: internal/tui/session/render_chat.go, internal/tui/viewport/window.go.
package session

import "github.com/keonho-kim/meca/internal/tui/viewport"

type chatProjection struct {
	thinking, stream, status, pending viewport.Block
	layout                            viewport.Transcript
}

func (a *app) chatLayout(width int) *viewport.Transcript {
	p := &a.chatProjection
	p.thinking.Text, p.thinking.Thinking, p.thinking.Markdown = a.thinking, true, true
	p.stream.Text, p.stream.Markdown = a.stream, true
	p.status.Text = a.status
	pending := ""
	if len(a.pending) > 0 {
		pending = a.language.Format("Queued (%d) > ", len(a.pending)) + a.opts.Vault.Redact(a.pending[0])
	}
	p.pending.Text, p.pending.Thinking = pending, true
	next := a.output.Next + 1
	live := []*viewport.Block{&p.thinking, &p.stream, &p.status, &p.pending}
	for _, block := range live {
		block.ID = next
		if block.Count(width) > 0 {
			next++
		}
	}
	p.layout.Prepare(width, a.output.Blocks, live)
	return &p.layout
}
