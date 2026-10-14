// Purpose: Load older ordered terminal records on demand while keeping model context separate.
// Pattern: Presentation projection over paged repository data.
// Usage: Called at restoration and when scrolling beyond the oldest loaded output.
// Related: internal/storage/sqlite/timeline.go, internal/tui/viewport/viewport.go, internal/tui/session/keys.go, internal/tui/session/events.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/tui/transcript"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

func (a *app) loadHistory(older bool) error {
	records, cursor, err := a.opts.Store.Timeline(a.ctx, a.opts.Session.ID, a.historyCursor, 100)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	a.historyCursor = cursor
	var blocks []viewport.Block
	formatter := transcript.Formatter{Redact: a.opts.Vault.Redact}
	for _, record := range records {
		if record.Kind == "message" {
			blocks = append(blocks, formatter.Message(record.Message)...)
		} else {
			blocks = append(blocks, formatter.Event(record.Event)...)
		}
	}
	if older {
		a.output.Prepend(blocks)
	} else {
		for _, block := range blocks {
			a.output.Append(block.Text, block.Thinking)
		}
	}
	return nil
}
func (a *app) scroll(n, w, h int) {
	a.output.Scroll(n, w, h)
	if n < 0 && !a.output.Follow && a.output.Start(a.output.Lines(w), h) == 0 {
		a.fail(a.loadHistory(true))
	}
}
