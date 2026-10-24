// Purpose: Project ordered persisted records into localized conversation blocks without changing literal content.
// Pattern: Presentation projection.
// Usage: Session restoration and older-history workers share Formatter.History.
// Related: internal/storage/history/record.go, internal/tui/transcript/messages.go, internal/tui/transcript/events.go.
package transcript

import (
	"github.com/keonho-kim/meca/internal/storage/history"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (f Formatter) History(records []history.Record) []viewport.Block {
	var blocks []viewport.Block
	for _, record := range records {
		if record.Kind == "message" {
			blocks = append(blocks, f.Message(record.Message)...)
		} else {
			blocks = append(blocks, f.Event(record.Event)...)
		}
	}
	return blocks
}
