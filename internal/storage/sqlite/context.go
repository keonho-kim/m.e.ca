// Purpose: Load every uncompacted model message in chronological order without a newest-record cutoff.
// Pattern: Repository projection.
// Usage: The agent journal supplies a committed compaction timeline boundary and a byte budget.
// Related: internal/storage/sqlite/messages.go, internal/storage/sqlite/timeline.go, internal/agent/run/context.go.
package sqlite

import (
	"bytes"
	"context"
	"fmt"

	json "github.com/goccy/go-json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func (s *Store) ContextMessages(ctx context.Context, session string, after int64, maxBytes int, images bool) (messages []modelapi.Message, err error) {
	if after < 0 || maxBytes < 1 {
		return nil, fmt.Errorf("invalid conversation context bounds")
	}
	page, err := s.recordPage(ctx, `SELECT t.sequence,t.kind,m.body,m.refs,m.raw_bytes FROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence>? ORDER BY t.sequence`, session, after)
	if err != nil {
		return nil, err
	}
	total := 0
	for _, record := range page {
		var fields map[string]bool
		if !images {
			fields, err = textContextFields(record.body)
			if err != nil {
				return nil, err
			}
		}
		if fields == nil && total+record.size > maxBytes {
			return nil, fmt.Errorf("uncompacted conversation exceeds %d bytes; no messages were omitted", maxBytes)
		}
		body, err := s.unpackRecord(ctx, record, fields)
		if err != nil {
			return nil, err
		}
		var message modelapi.Message
		if err := json.Unmarshal(body, &message); err != nil {
			return nil, err
		}
		size := record.size
		if fields != nil {
			message.Content += modelapi.ImageOmissionNotice
			size = len(body) + len(modelapi.ImageOmissionNotice)
		}
		total += size
		if total > maxBytes {
			return nil, fmt.Errorf("uncompacted conversation exceeds %d bytes; no messages were omitted", maxBytes)
		}
		if err := message.Check(); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
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
		if err := json.Unmarshal(raw, &pictures); err != nil {
			return nil, err
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
