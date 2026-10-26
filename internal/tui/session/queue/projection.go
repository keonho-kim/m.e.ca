// Purpose: Cache compact reservation rows and release source/display references when requests leave the queue.
// Pattern: Event-loop-owned queue presentation projection.
// Usage: Chat layout combines Header and Entries with history and other live blocks.
// Related: internal/tui/session/queue/state.go, internal/tui/session/chat_projection.go.
package queue

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/viewport"
	"slices"
	"strings"
)

type Projection struct {
	Header           viewport.Block
	Entries          []projectionEntry
	redactionVersion int
}

type projectionEntry struct {
	source string
	Block  viewport.Block
}

func (p *Projection) Prepare(messages []string, language locale.Language, version int, redact func(string) string) {
	p.Header.Text, p.Header.Thinking = "", true
	if len(messages) == 0 {
		clear(p.Entries)
		p.Entries = nil
		return
	}
	p.Header.Text = language.Format("Queued (%d)", len(messages))
	if len(p.Entries) > len(messages) {
		clear(p.Entries[len(messages):])
		p.Entries = p.Entries[:len(messages)]
	} else if len(p.Entries) < len(messages) {
		p.Entries = slices.Grow(p.Entries, len(messages)-len(p.Entries))[:len(messages)]
	}
	for i, text := range messages {
		entry := &p.Entries[i]
		if entry.source != text || entry.Block.Text == "" || p.redactionVersion != version {
			entry.source = text
			prefix := fmt.Sprintf("  %d. ", i+1)
			body := strings.ReplaceAll(redact(text), "\n", "\n"+strings.Repeat(" ", len(prefix)))
			entry.Block.Text = prefix + body
			entry.Block.Thinking = true
		}
	}
	p.redactionVersion = version
}
