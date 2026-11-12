// Purpose: Project persisted records into mutually exclusive transcript entries.
// Pattern: Pure read adapter with durable timeline identity.
// Usage: All history transports share this conversion.
// Related: internal/serverapi/event_content.go, internal/serverapi/history_completion.go.
package serverapi

import (
	"strings"

	"github.com/keonho-kim/meca/internal/commands/server"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func historyItem(value history.Record) *v1.HistoryItem {
	item := &v1.HistoryItem{Sequence: value.Sequence, Truncated: value.Truncated}
	if completion := historyCompletion(value.Event); value.Kind == "event" && completion != nil {
		item.Entry = &v1.HistoryItem_Completion{Completion: completion}
		return item
	}
	if value.Kind == "message" {
		m := &v1.TranscriptMessage{Role: value.Message.Role, Text: strings.Clone(textPreview(value.Message.Content)), Incomplete: value.Message.Incomplete}
		item.Truncated = item.Truncated || len(m.Text) < len(value.Message.Content)
		for index, image := range value.Images {
			m.Images = append(m.Images, wireImage(index, image))
		}
		item.Entry = &v1.HistoryItem_Message{Message: m}
	} else {
		value.Event.HistorySequence = value.Sequence
		item.Entry = &v1.HistoryItem_Event{Event: event(server.ProjectEvent(server.Event{Record: value.Event}))}
	}
	return item
}
