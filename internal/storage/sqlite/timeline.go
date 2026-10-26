// Purpose: Page ordered message and tool-event references for terminal history without duplicating bodies.
// Pattern: Repository projection.
// Usage: Used by session restoration and scrolling into older history.
// Related: internal/storage/sqlite/timeline_test.go, internal/tui/session/history.go.
package sqlite

import (
	"context"
	"fmt"

	json "github.com/goccy/go-json"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) Timeline(ctx context.Context, session string, before int64, limit int) (records []history.Record, cursor int64, err error) {
	if limit < 1 || limit > 200 {
		return nil, 0, fmt.Errorf("invalid history page")
	}
	if before == 0 {
		before = 1 << 62
	}
	page, err := s.recordPage(ctx, `SELECT t.sequence,t.kind,COALESCE(m.body,e.body),COALESCE(m.refs,e.refs),COALESCE(m.raw_bytes,e.raw_bytes) FROM timeline t LEFT JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence LEFT JOIN tool_events e ON t.kind='event' AND t.source_sequence=e.sequence WHERE t.session_id=? AND t.sequence<? ORDER BY t.sequence DESC LIMIT ?`, session, before, limit)
	if err != nil {
		return nil, 0, err
	}
	bytes := 0
	for _, record := range page {
		var fields map[string]bool
		if record.kind == "event" {
			fields = map[string]bool{}
			var root map[string]json.RawMessage
			if err = json.Unmarshal(record.body, &root); err != nil {
				return nil, 0, err
			}
			for key := range root {
				if key != "capture" {
					fields[key] = true
				}
			}
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
