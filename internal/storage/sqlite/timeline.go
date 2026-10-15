// Purpose: Page ordered message and tool-event references for terminal history without duplicating bodies.
// Pattern: Repository projection.
// Usage: Used by session restoration and scrolling into older history.
// Related: internal/storage/sqlite/timeline_test.go, internal/tui/session/history.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/storage/history"
)

func (s *Store) Timeline(ctx context.Context, session string, before int64, limit int) (records []history.Record, cursor int64, err error) {
	if limit < 1 || limit > 200 {
		return nil, 0, fmt.Errorf("invalid history page")
	}
	if before == 0 {
		before = 1 << 62
	}
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,t.kind,CASE t.kind WHEN 'message' THEN m.body ELSE e.body END FROM timeline t LEFT JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence LEFT JOIN tool_events e ON t.kind='event' AND t.source_sequence=e.sequence WHERE t.session_id=? AND t.sequence<? ORDER BY t.sequence DESC LIMIT ?`, session, before, limit)
	if err != nil {
		return nil, 0, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	bytes := 0
	for rows.Next() {
		var r history.Record
		var body []byte
		if err = rows.Scan(&r.Sequence, &r.Kind, &body); err != nil {
			return nil, 0, err
		}
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
	return records, cursor, rows.Err()
}
