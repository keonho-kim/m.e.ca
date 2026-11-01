// Purpose: Project ordered persisted records into localized conversation blocks without changing literal content.
// Pattern: Presentation projection.
// Usage: Session restoration and older-history workers share Formatter.History.
// Related: internal/storage/history/record.go, internal/tui/transcript/messages.go, internal/tui/transcript/events.go.
package transcript

import (
	"strings"

	"github.com/keonho-kim/meca/internal/storage/history"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (f Formatter) History(records []history.Record) []viewport.Block {
	var blocks []viewport.Block
	var lastGroup viewport.Group
	for _, record := range records {
		start := len(blocks)
		if record.Kind == "message" {
			blocks = append(blocks, f.Message(record.Message)...)
		} else {
			blocks = append(blocks, f.Event(record.Event)...)
		}
		for i := start; i < len(blocks); i++ {
			block := &blocks[i]
			if !record.Event.Internal && strings.HasPrefix(record.Event.Kind, "lat_") &&
				record.Event.Kind != "lat_memory_unavailable" &&
				lastGroup.ID != "" && strings.HasPrefix(lastGroup.ID, record.Event.RunID+":") {
				block.Group = lastGroup
			}
			lastGroup = block.Group
		}
	}
	return blocks
}
