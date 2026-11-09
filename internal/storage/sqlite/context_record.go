// Purpose: Restore model-message metadata and already authenticated shared content without a JSON body copy.
// Pattern: Page-scoped message projection.
// Usage: ContextMessages and MemoryRecords reuse the same shared-content admission.
// Related: internal/storage/sqlite/context.go, internal/storage/sqlite/team_context_page.go.
package sqlite

import (
	"context"

	json "github.com/goccy/go-json"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) contextRecord(ctx context.Context, record storedRecord, fields map[string]bool, shared map[string]string) (modelapi.Message, int, error) {
	var refs map[string]string
	if len(shared) > 0 {
		if err := json.Unmarshal(record.refs, &refs); err != nil {
			return modelapi.Message{}, 0, err
		}
	}
	content, sharedBody := shared[refs[`["content"]`]]
	sharedBody = sharedBody && len(refs) == 1
	body := record.body
	var err error
	if !sharedBody {
		body, err = s.unpackRecord(ctx, record, fields)
		if err != nil {
			return modelapi.Message{}, 0, err
		}
	} else if fields != nil {
		body, err = recordcontent.Restore(body, []byte("{}"), fields, nil)
		if err != nil {
			return modelapi.Message{}, 0, err
		}
	}
	var message modelapi.Message
	if err := json.Unmarshal(body, &message); err != nil {
		return message, 0, err
	}
	if sharedBody {
		var metadata struct {
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(body, &metadata); err != nil {
			return message, 0, err
		}
		if string(metadata.Content) != "null" {
			return message, 0, faults.Errorf(diagcatalog.SqliteInvalidSharedContentPlaceholder)
		}
		message.Content = content
	}
	size := len(body)
	if sharedBody && record.size > 0 {
		size += record.size - len(record.body)
	}
	return message, size, nil
}
