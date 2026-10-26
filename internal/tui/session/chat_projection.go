// Purpose: Compose history and live blocks into one sparse, event-loop-owned transcript index.
// Pattern: Presentation projection.
// Usage: Chat rendering, scrolling and text selection share chatLayout.
// Related: internal/tui/session/render_chat.go, internal/tui/viewport/window.go.
package session

import "github.com/keonho-kim/meca/internal/tui/viewport"

type chatProjection struct {
	thinking, stream, status, pending viewport.Block
	layout                            viewport.Transcript
	queued                            []pendingBlock
	live                              []*viewport.Block
	redactionVersion, focusedID       int
}

func (a *app) chatLayout(width int) *viewport.Transcript {
	p := &a.chatProjection
	queueAnchor := -1
	if !a.output.Follow && p.pending.Text != "" {
		offset := a.output.AnchorID - p.pending.ID
		if offset >= 0 && offset <= len(p.queued) {
			queueAnchor = offset
		}
	}
	p.thinking.Text, p.thinking.Thinking, p.thinking.Markdown = a.thinking, true, true
	p.stream.Text, p.stream.Markdown = a.stream, true
	p.status.Text = a.status
	a.projectPending()
	next := a.output.Next + 1
	clear(p.live)
	p.live = append(p.live[:0], &p.thinking, &p.stream, &p.status, &p.pending)
	for i := range p.queued {
		p.live = append(p.live, &p.queued[i].block)
	}
	for _, block := range p.live {
		block.ID = next
		if block.Count(width) > 0 {
			next++
		}
	}
	p.focusedID = 0
	if a.pendingEdit != nil {
		p.focusedID = p.queued[a.pendingEdit.index].block.ID
	}
	if queueAnchor >= 0 && len(p.queued) > 0 {
		// History and live blocks precede these contiguous queue IDs and may grow.
		a.output.AnchorID = p.pending.ID + min(queueAnchor, len(p.queued))
	}
	p.layout.Prepare(width, a.output.Blocks, p.live)
	return &p.layout
}
