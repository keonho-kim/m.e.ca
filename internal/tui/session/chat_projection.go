// Purpose: Compose history and live blocks into one sparse, event-loop-owned transcript index.
// Pattern: Presentation projection.
// Usage: Chat rendering, scrolling and text selection share chatLayout.
// Related: internal/tui/session/render_chat.go, internal/tui/viewport/window.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/composer/queue"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type chatProjection struct {
	thinking, stream, tools, status viewport.Block
	layout                          viewport.Transcript
	queue                           queue.Projection
	live                            []*viewport.Block
	focusedID                       int
}

func (a *app) chatLayout(width int) *viewport.Transcript {
	p := &a.chatProjection
	queueAnchor := -1
	if !a.output.Follow && p.queue.Header.Text != "" {
		offset := a.output.AnchorID - p.queue.Header.ID
		if offset >= 0 && offset <= len(p.queue.Entries) {
			queueAnchor = offset
		}
	}
	p.thinking.Text, p.thinking.Thinking, p.thinking.Markdown = a.thinking, true, true
	p.stream.Text, p.stream.Markdown = a.stream, true
	p.tools.Text = a.liveToolActivity()
	p.status.Text = a.status
	p.queue.Prepare(a.queued.Messages, a.language, a.opts.Vault.RedactionVersion(), a.opts.Vault.Redact)
	next := a.output.Next + 1
	clear(p.live)
	p.live = append(p.live[:0], &p.thinking, &p.stream, &p.tools, &p.status, &p.queue.Header)
	for i := range p.queue.Entries {
		p.live = append(p.live, &p.queue.Entries[i].Block)
	}
	for _, block := range p.live {
		block.ID = next
		if block.Count(width) > 0 {
			next++
		}
	}
	p.focusedID = 0
	if a.queued.Edit != nil {
		p.focusedID = p.queue.Entries[a.queued.Edit.Index].Block.ID
	}
	if queueAnchor >= 0 && len(p.queue.Entries) > 0 {
		// History and live blocks precede these contiguous queue IDs and may grow.
		a.output.AnchorID = p.queue.Header.ID + min(queueAnchor, len(p.queue.Entries))
	}
	p.layout.Prepare(width, &a.output, p.live)
	return &p.layout
}
