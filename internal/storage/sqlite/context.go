// Purpose: Load every uncompacted model message in chronological order without a newest-record cutoff.
// Pattern: Repository projection.
// Usage: The agent journal supplies a committed compaction timeline boundary and a byte budget.
// Related: internal/storage/sqlite/messages.go, internal/agent/run/context.go.
package sqlite

import (
	"bytes"
	"context"

	json "github.com/goccy/go-json"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) ContextMessages(ctx context.Context, session string, after int64, maxBytes int, images bool, reserve func(int) error) (messages []modelapi.Message, err error) {
	if err := s.checkSession(session); err != nil {
		return nil, err
	}
	if after < 0 || maxBytes < 1 {
		return nil, faults.Errorf(diagcatalog.SqliteInvalidConversationContextBounds)
	}
	total := 0
	for {
		page, err := s.recordPage(ctx, "SELECT t.sequence,t.kind,m.body,m.refs,m.raw_bytes FROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence WHERE t.sequence>? ORDER BY t.sequence LIMIT 64", after)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return messages, nil
		}
		needed := total
		selectedFields := make([]map[string]bool, len(page))
		for i, record := range page {
			if !images {
				selectedFields[i], err = textContextFields(record.body)
				if err != nil {
					return nil, err
				}
			}

		}
		sizes, sizeErr := s.contextReadSizes(ctx, page, selectedFields)
		if sizeErr != nil {
			return nil, sizeErr
		}
		for _, size := range sizes {
			if size > maxBytes-needed {
				return nil, faults.Errorf(diagcatalog.SqliteUncompactedConversationExceedsBytesNoMessagesWere, maxBytes)
			}
			needed += size
		}
		if reserve != nil {
			if err := reserve(needed); err != nil {
				return nil, err
			}
		}

		var shared map[string]string
		if s.room != nil {
			selected := page
			size := total
			for i, record := range page {
				size += record.size
				if size > maxBytes {
					selected = page[:i]
					break
				}
			}
			shared, err = s.sharedContext(ctx, selected)
			if err != nil {
				return nil, err
			}
		}
		for i, record := range page {
			fields := selectedFields[i]
			if fields == nil && total+record.size > maxBytes {
				return nil, faults.Errorf(diagcatalog.SqliteUncompactedConversationExceedsBytesNoMessagesWere, maxBytes)
			}

			message, bodyBytes, err := s.contextRecord(ctx, record, fields, shared)
			if err != nil {
				return nil, err
			}
			size := record.size
			if fields != nil {
				message.Content += modelapi.ImageOmissionNotice
				size = bodyBytes + len(modelapi.ImageOmissionNotice)
			}
			total += size
			if total > maxBytes {
				return nil, faults.Errorf(diagcatalog.SqliteUncompactedConversationExceedsBytesNoMessagesWere, maxBytes)
			}
			if err := message.Check(); err != nil {
				return nil, err
			}
			messages = append(messages, message)
			after = record.sequence
		}
	}
}

func textContextFields(body []byte) (map[string]bool, error) {
	// The fast path does not parse ordinary text records a second time. A possible
	// key match is always confirmed structurally before any field is excluded.
	if !bytes.Contains(body, []byte(`"images"`)) {
		return nil, nil
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	var pictures []struct{}
	if raw := root["images"]; raw != nil {
		if string(raw) == "null" {
			pictures = make([]struct{}, 1)
		} else {
			if err := json.Unmarshal(raw, &pictures); err != nil {
				return nil, err
			}
		}
	}
	if len(pictures) == 0 {
		return nil, nil
	}
	fields := make(map[string]bool, len(root)-1)
	for key := range root {
		if key != "images" {
			fields[key] = true
		}
	}
	return fields, nil
}
