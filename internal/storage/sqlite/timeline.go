// Purpose: Page ordered message and tool-event references for terminal history without duplicating bodies.
// Pattern: Repository projection.
// Usage: Used by session restoration and scrolling into older history.
// Related: internal/storage/sqlite/timeline_test.go, internal/tui/session/history.go.
package sqlite

import (
	"context"
	"fmt"

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
	payloads := pagePayloads{store: s, ctx: ctx}
	for i, entry := range page {
		if err := payloads.include(i, entry.record.refs); err != nil {
			return nil, 0, err
		}
	}
	for i, entry := range page {
		record := entry.record
		// Page acceptance precedes payload hydration. An individually large
		// visible record remains readable, but cannot overfill a preceding page.
		if bytes+record.size > 16*1024*1024 && len(records) > 0 {
			break
		}
		fields, nested := history.DisplayFields(record.kind, entry.internal)
		body, decodeErr := payloads.unpack(i, record, fields, nested)
		if decodeErr != nil {
			return nil, 0, decodeErr
		}
		bytes += len(body)
		if bytes > 16*1024*1024 && len(records) > 0 {
			break
		}
		cursor = record.sequence
		r, err := history.Display(record.sequence, record.kind, body, record.body, entry.arguments)
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
