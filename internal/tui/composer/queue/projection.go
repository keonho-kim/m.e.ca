// Purpose: Cache compact reservation rows and release source/display references when requests leave the queue.
// Pattern: Event-loop-owned queue presentation projection.
// Usage: Chat layout combines Header and Entries with history and other live blocks.
// Related: internal/tui/composer/queue/state.go, internal/tui/session/chat_projection.go.
package queue

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/viewport"
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

func (p *Projection) Prepare(messages []requestinput.Request, language locale.Language, version int, redact func(string) string) {
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
	for i, request := range messages {
		text := request.Text
		entry := &p.Entries[i]
		if entry.source != text || entry.Block.Text == "" || p.redactionVersion != version {
			entry.source = text
			prefix := fmt.Sprintf("  %d. ", i+1)
			body := redact(text)
			end := min(len(body), 2048)
			offset := 0
			for range 12 {
				next := strings.IndexByte(body[offset:end], '\n')
				if next < 0 {
					break
				}
				offset += next + 1
			}
			if offset > 0 && strings.Count(body[:offset], "\n") == 12 {
				end = offset
			}
			if end < len(body) {
				for end > 0 && !utf8.RuneStart(body[end]) {
					end--
				}
				body = body[:end] + "\n" + language.Text("[Preview; edit the queued request to read the complete text]")
			}
			body = strings.ReplaceAll(body, "\n", "\n"+strings.Repeat(" ", len(prefix)))
			entry.Block.Text = prefix + body
			entry.Block.Thinking = true
		}
	}
	p.redactionVersion = version
}
