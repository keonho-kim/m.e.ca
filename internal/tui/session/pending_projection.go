// Purpose: Project every queued request as a reusable dim block and reveal keyboard focus.
// Pattern: Event-loop-owned presentation projection.
// Usage: chatLayout prepares queue blocks; renderChat reveals newly selected items.
// Related: internal/tui/session/chat_projection.go, internal/tui/session/pending_edit.go, internal/tui/viewport/window.go.
package session

import (
	"fmt"
	"slices"
	"strings"

	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type pendingBlock struct {
	source string
	block  viewport.Block
}

func (a *app) projectPending() {
	p := &a.chatProjection
	p.pending.Text, p.pending.Thinking = "", true
	if len(a.pending) == 0 {
		clear(p.queued)
		p.queued = nil
		return
	}
	p.pending.Text = a.language.Format("Queued (%d)", len(a.pending))
	if len(p.queued) > len(a.pending) {
		clear(p.queued[len(a.pending):])
		p.queued = p.queued[:len(a.pending)]
	} else if len(p.queued) < len(a.pending) {
		p.queued = slices.Grow(p.queued, len(a.pending)-len(p.queued))[:len(a.pending)]
	}
	version := a.opts.Vault.RedactionVersion()
	for i, text := range a.pending {
		entry := &p.queued[i]
		if entry.source != text || entry.block.Text == "" || p.redactionVersion != version {
			entry.source = text
			prefix := fmt.Sprintf("  %d. ", i+1)
			body := strings.ReplaceAll(a.opts.Vault.Redact(text), "\n", "\n"+strings.Repeat(" ", len(prefix)))
			entry.block.Text = prefix + body
			entry.block.Thinking = true
		}
	}
	p.redactionVersion = version
}

func (a *app) revealPending(transcript *viewport.Transcript, height int) {
	edit := a.pendingEdit
	if edit == nil || !edit.reveal || height < 1 {
		return
	}
	anchor := viewport.View{AnchorID: a.chatProjection.focusedID}
	target := transcript.Start(&anchor, height)
	start := transcript.Start(&a.output, height)
	if target < start || target >= start+height {
		transcript.Seek(&a.output, target, height)
	}
	edit.reveal = false
}
