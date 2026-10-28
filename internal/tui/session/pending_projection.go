// Purpose: Keep the currently selected reservation visible without changing deliberate history scrolling.
// Pattern: Event-loop viewport coordination.
// Usage: Chat rendering reveals newly selected or resized reservation rows.
// Related: internal/tui/composer/queue/projection.go, internal/tui/session/render_chat.go.
package session

import "github.com/keonho-kim/meca/internal/tui/viewport"

func (a *app) revealPending(transcript *viewport.Transcript, height int) {
	edit := a.queued.Edit
	if edit == nil || !edit.Reveal || height < 1 {
		return
	}
	anchor := viewport.View{AnchorID: a.chatProjection.focusedID}
	target := transcript.Start(&anchor, height)
	start := transcript.Start(&a.output, height)
	if target < start || target >= start+height {
		transcript.Seek(&a.output, target, height)
	}
	edit.Reveal = false
}
