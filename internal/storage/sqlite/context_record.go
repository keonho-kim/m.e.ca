// Purpose: Restore model messages and project verified single text payloads directly into their records.
// Pattern: Page-scoped message projection.
// Usage: ContextMessages and MemoryRecords share field selection and payload admission.
// Related: internal/storage/sqlite/context.go, internal/storage/sqlite/payload.go, internal/storage/sqlite/team_context_page.go.
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
	if fields == nil && s.room != nil {
		if message, ok, err := s.roomImageContext(ctx, record); ok || err != nil {
			return message, record.size, err
		}
	}
	localContent := s.room == nil && (fields == nil || fields["content"])
	if len(shared) > 0 || localContent && len(record.refs) > 2 {
		var refs map[string]string
		if err := json.Unmarshal(record.refs, &refs); err != nil {
			return modelapi.Message{}, 0, err
		}
		if len(refs) == 1 {
			reference := refs[`["content"]`]
			if content, ok := shared[reference]; ok {
				addedBytes := 0
				if record.size > 0 {
					addedBytes = record.size - len(record.body)
				}
				return contextRecordWithContent(record.body, fields, content, addedBytes)
			}
			if id, quoted := recordcontent.Reference(reference); localContent && id != "" && !quoted {
				payload, err := s.payload(ctx, id)
				if err != nil {
					return modelapi.Message{}, 0, err
				}
				var text string
				if err := json.Unmarshal(payload, &text); err != nil {
					return modelapi.Message{}, 0, err
				}
				return contextRecordWithContent(record.body, fields, text, len(payload)-len("null"))
			}
		}
	}
	body, err := s.unpackRecord(ctx, record, fields)
	if err != nil {
		return modelapi.Message{}, 0, err
	}
	var message modelapi.Message
	if err := json.Unmarshal(body, &message); err != nil {
		return message, 0, err
	}
	return message, len(body), nil
}

func contextRecordWithContent(body []byte, fields map[string]bool, content string, addedBytes int) (modelapi.Message, int, error) {
	if fields != nil {
		var err error
		body, err = recordcontent.Restore(body, []byte("{}"), fields, nil)
		if err != nil {
			return modelapi.Message{}, 0, err
		}
	}
	var metadata struct {
		modelapi.Message
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(body, &metadata); err != nil {
		return modelapi.Message{}, 0, err
	}
	if len(metadata.Content) == 0 {
		return modelapi.Message{}, 0, faults.Errorf(diagcatalog.ContentMissingStoredPayloadField)
	}
	if string(metadata.Content) != "null" {
		return modelapi.Message{}, 0, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadPlaceholder)
	}
	metadata.Message.Content = content
	return metadata.Message, len(body) + addedBytes, nil
}
