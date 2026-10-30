// Purpose: Retain complete recent native message groups while selecting an older compaction prefix.
// Pattern: Paged repository projection.
// Usage: Compaction reads metadata pages and hydrates only the recent messages needed for its tail.
// Related: internal/storage/history/boundary.go, internal/storage/sqlite/record_page.go.
package sqlite

import (
	"context"
	"fmt"

	json "github.com/goccy/go-json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) CompactionBoundary(ctx context.Context, session string, through int64, keepBytes int) (boundary int64, err error) {
	if through < 0 || keepBytes < 0 {
		return 0, fmt.Errorf("invalid compaction tail bounds")
	}
	pin, pinErr := s.unreadImageBoundary(ctx, session, through)
	if pinErr != nil {
		return 0, pinErr
	}
	defer func() {
		if err == nil {
			boundary = min(boundary, pin)
		}
	}()
	if keepBytes == 0 {
		return through, nil
	}
	selection := history.TailSelection{Target: keepBytes}
	before := through + 1
	for {
		page, err := s.recordPage(ctx, `SELECT t.sequence,t.kind,m.body,m.refs,m.raw_bytes FROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence<? ORDER BY t.sequence DESC LIMIT 64`, session, before)
		if err != nil {
			return 0, err
		}
		if len(page) == 0 {
			return 0, nil
		}
		for _, record := range page {
			body, err := s.unpackRecord(ctx, record, map[string]bool{"role": true, "content": true, "calls": true, "tool_call_id": true, "tool_name": true, "incomplete": true})
			if err != nil {
				return 0, err
			}
			var message modelapi.Message
			if err := json.Unmarshal(body, &message); err != nil {
				return 0, err
			}
			if boundary, ready := selection.Keep(record.sequence, record.size, message); ready {
				return boundary, nil
			}
			before = record.sequence
		}
	}
}
