// Purpose: Read compact observations directly and hydrate only message content for summarization.
// Pattern: SQLite projection adapter.
// Usage: Compaction pages avoid large reasoning, tool arguments and original captures.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/record_codec.go.
package sqlite

import (
	"context"
	"errors"

	json "github.com/goccy/go-json"
	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) MemoryRecords(ctx context.Context, session string, after, through int64) (records []compact.Record, cursor int64, err error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,t.kind,m.body,m.refs,e.memory_role,e.memory_text,COALESCE(m.content_bytes,0) FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence>? AND t.sequence<=? ORDER BY t.sequence LIMIT 64`, session, after, through)
	if err != nil {
		return nil, after, err
	}
	type entry struct {
		record     storedRecord
		role, text string
	}
	var page []entry
	bytes := 0
	cursor = after
	for rows.Next() {
		var e entry
		var role, text *string
		var contentBytes int
		if err = rows.Scan(&e.record.sequence, &e.record.kind, &e.record.body, &e.record.refs, &role, &text, &contentBytes); err != nil {
			break
		}
		if role != nil {
			e.role = *role
		}
		if text != nil {
			e.text = *text
		}
		bytes += contentBytes + len(e.text)
		if bytes > 16*1024*1024 && cursor > after {
			break
		}
		page = append(page, e)
		cursor = e.record.sequence
	}
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil {
		return nil, cursor, err
	}
	for _, e := range page {
		if e.record.kind == "message" {
			body, err := s.unpackRecord(ctx, e.record, map[string]bool{"role": true, "content": true, "incomplete": true})
			if err != nil {
				return nil, cursor, err
			}
			var message modelapi.Message
			if err := json.Unmarshal(body, &message); err != nil {
				return nil, cursor, err
			}
			projected, err := history.Compaction([]history.Record{{Sequence: e.record.sequence, Kind: "message", Message: message}})
			if err != nil {
				return nil, cursor, err
			}
			records = append(records, projected...)
		} else if e.role != "" && e.text != "" {
			records = append(records, compact.Record{Sequence: e.record.sequence, Role: e.role, Text: e.text})
		}
	}
	return records, cursor, nil
}
