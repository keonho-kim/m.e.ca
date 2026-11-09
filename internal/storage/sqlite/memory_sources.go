// Purpose: Read compact observations directly and hydrate only message content for summarization.
// Pattern: SQLite projection adapter.
// Usage: Compaction pages avoid large reasoning, tool arguments and original captures.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/record_codec.go.
package sqlite

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/agent/compact"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) MemoryRecords(ctx context.Context, session string, after, through int64) (records []compact.Record, cursor int64, err error) {
	if err := s.checkSession(session); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT t.sequence,t.kind,m.body,m.refs,e.memory_role,e.memory_text,COALESCE(m.content_bytes,e.memory_bytes,0),COALESCE(e.memory_source,0) FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence WHERE t.sequence>? AND t.sequence<=? ORDER BY t.sequence LIMIT 64", after, through)
	if err != nil {
		return nil, after, err
	}
	type entry struct {
		record     storedRecord
		role, text string
		public     int64
	}
	var page []entry
	bytes := 0
	cursor = after
	for rows.Next() {
		var e entry
		var role, text *string
		var contentBytes int
		if err = rows.Scan(&e.record.sequence, &e.record.kind, &e.record.body, &e.record.refs, &role, &text, &contentBytes, &e.public); err != nil {
			break
		}
		if role != nil {
			e.role = *role
		}
		if text != nil {
			e.text = *text
		}

		bytes += contentBytes
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
	var shared map[string]string
	if s.room != nil {
		messages := make([]storedRecord, 0, len(page))
		for _, e := range page {
			if e.record.kind == "message" {
				messages = append(messages, e.record)
			}
		}
		shared, err = s.sharedContext(ctx, messages)
		if err != nil {
			return nil, cursor, err
		}
	}
	fields := map[string]bool{"source": true, "id": true, "role": true, "content": true, "incomplete": true}
	var sequences []any
	for _, e := range page {
		if e.public > 0 {
			sequences = append(sequences, e.public)
		}
	}
	var authored map[int64]string
	if len(sequences) > 0 {
		authored, err = s.publicContexts(ctx, sequences)
		if err != nil {
			return nil, cursor, err
		}
	}
	for _, e := range page {
		if e.record.kind == "message" {
			message, _, err := s.contextRecord(ctx, e.record, fields, shared)
			if err != nil {
				return nil, cursor, err
			}
			projected, err := history.Compaction([]history.Record{{Sequence: e.record.sequence, Kind: "message", Message: message}})
			if err != nil {
				return nil, cursor, err
			}
			records = append(records, projected...)
		} else if e.public > 0 {
			text, ok := authored[e.public]
			if !ok {
				return nil, cursor, faults.Errorf(diagcatalog.SqliteAuthoredPublicNoteIsMissing, e.public)
			}
			records = append(records, compact.Record{Sequence: e.record.sequence, Role: "peer", Text: text})
		} else if e.role != "" && e.text != "" {
			records = append(records, compact.Record{Sequence: e.record.sequence, Role: e.role, Text: e.text})
		}
	}
	return records, cursor, nil
}
