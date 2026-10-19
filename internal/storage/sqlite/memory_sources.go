// Purpose: Page and decode chronological history for the storage compaction projection.
// Pattern: SQLite adapter.
// Usage: The compaction repository reads bounded pages through MemoryRecords.
// Related: internal/storage/history/compaction.go, internal/storage/sqlite/memory.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/storage/history"
)

func (s *Store) MemoryRecords(ctx context.Context, session string, after, through int64) (records []compact.Record, cursor int64, err error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,t.kind,COALESCE(json_extract(m.body,'$.role'),''),COALESCE(json_extract(m.body,'$.content'),''),json_remove(e.body,'$.capture') FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence>? AND t.sequence<=? ORDER BY t.sequence LIMIT 64`, session, after, through)
	if err != nil {
		return nil, after, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	var page []history.Record
	bytes := 0
	cursor = after
	for rows.Next() {
		var sequence int64
		var kind, role, text string
		var body []byte
		if err = rows.Scan(&sequence, &kind, &role, &text, &body); err != nil {
			return nil, cursor, err
		}
		bytes += len(body) + len(text)
		if bytes > 16*1024*1024 && cursor > after {
			break
		}
		cursor = sequence
		record := history.Record{Sequence: sequence, Kind: kind}
		if kind == "message" {
			record.Message.Role, record.Message.Content = role, text
		} else if err := json.Unmarshal(body, &record.Event); err != nil {
			return nil, cursor, err
		}
		page = append(page, record)
	}
	if err := rows.Err(); err != nil {
		return nil, cursor, err
	}
	records, err = history.Compaction(page)
	return records, cursor, err
}
