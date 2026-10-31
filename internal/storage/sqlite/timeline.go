// Purpose: Page ordered message and tool-event references for terminal history without duplicating bodies.
// Pattern: Repository projection.
// Usage: Used by session restoration and scrolling into older history.
// Related: internal/storage/sqlite/timeline_test.go, internal/tui/session/history.go.
package sqlite

import (
	"context"
	"errors"
	"fmt"

	json "github.com/goccy/go-json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) Timeline(ctx context.Context, session string, before int64, limit int) (records []history.Record, cursor int64, err error) {
	if limit < 1 || limit > 200 {
		return nil, 0, fmt.Errorf("invalid history page")
	}
	if before == 0 {
		before = 1 << 62
	}
	page, err := s.displayPage(ctx, session, before, limit)
	if err != nil {
		return nil, 0, err
	}
	bytes := 0
	for _, entry := range page {
		record := entry.record
		// Page acceptance precedes payload hydration. An individually large
		// visible record remains readable, but cannot overfill a preceding page.
		if bytes+record.size > 16*1024*1024 && len(records) > 0 {
			break
		}
		fields := map[string]bool{"role": true, "content": true, "incomplete": true}
		if record.kind == "event" {
			fields = map[string]bool{"kind": true, "text": true, "thinking": true, "call_id": true, "output": true, "operation": true, "change_preview": true, "checkpoint": true, "statistics": true, "internal": true}
		}
		body, decodeErr := s.unpackRecord(ctx, record, fields)
		if decodeErr != nil {
			return nil, 0, decodeErr
		}
		r := history.Record{Sequence: record.sequence, Kind: record.kind}
		bytes += len(body)
		if bytes > 16*1024*1024 && len(records) > 0 {
			break
		}
		cursor = r.Sequence
		if r.Kind == "message" {
			err = json.Unmarshal(body, &r.Message)
			if err == nil {
				err = r.Message.Check()
			}
		} else {
			err = json.Unmarshal(body, &r.Event)
			if err == nil {
				var envelope struct{ Call *struct{ ID, Name string } }
				if err = json.Unmarshal(record.body, &envelope); err != nil {
					return nil, 0, err
				}
				if envelope.Call != nil {
					r.Event.Call = &toolapi.Call{ID: envelope.Call.ID, Name: envelope.Call.Name, Arguments: entry.arguments}
				}
			}
		}
		if err != nil {
			return nil, 0, err
		}
		records = append(records, r)
	}
	for l, r := 0, len(records)-1; l < r; l, r = l+1, r-1 {
		records[l], records[r] = records[r], records[l]
	}
	return records, cursor, nil
}

type displayRecord struct {
	record    storedRecord
	arguments []byte
}

func (s *Store) displayPage(ctx context.Context, session string, before int64, limit int) (page []displayRecord, err error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,t.kind,COALESCE(m.body,e.body),COALESCE(m.refs,e.refs),COALESCE(m.raw_bytes,e.raw_bytes),COALESCE(e.display_args,'{}')
FROM timeline t LEFT JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence
LEFT JOIN tool_events e ON t.kind='event' AND t.source_sequence=e.sequence
WHERE t.session_id=? AND t.sequence<? AND (
 (m.role IN ('user','assistant') AND m.content_bytes>0) OR
 (e.kind IN ('thinking','progress','lat_clean','lat_feedback','lat_unavailable','lat_discarded','lat_memory_unavailable','tool_succeeded','tool_failed','tool_interrupted','denied','command_finished','run_checkpoint') AND COALESCE(json_extract(e.body,'$.internal'),0)=0))
ORDER BY t.sequence DESC LIMIT ?`, session, before, limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var entry displayRecord
		r := &entry.record
		if err = rows.Scan(&r.sequence, &r.kind, &r.body, &r.refs, &r.size, &entry.arguments); err != nil {
			return nil, err
		}
		page = append(page, entry)
	}
	return page, rows.Err()
}
