// Purpose: Admit selected user-input envelopes before allocating their stored bodies.
// Pattern: Two-stage immutable SQLite projection.
// Usage: RecentInputs keeps small histories inexpensive while bounding restored text separately.
// Related: internal/storage/sqlite/input_history.go, internal/storage/sqlite/record_page.go.
package sqlite

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) recentInputPage(ctx context.Context, limit int, reserve func(int) error) ([]storedRecord, int, error) {
	const metadataFloor = 32 << 10
	if err := reserve(metadataFloor); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `WITH recent AS (
 SELECT sequence,length(body)+length(refs) AS bytes FROM messages WHERE role='user' ORDER BY sequence DESC LIMIT ?
), bounded AS (
 SELECT *,SUM(bytes) OVER (ORDER BY sequence DESC) AS charged FROM recent
) SELECT sequence,bytes FROM bounded WHERE charged<=? ORDER BY sequence DESC`, limit, history.RecentInputBytes)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]any, 0, limit)
	total := 0
	for rows.Next() {
		var sequence int64
		var size int
		if err = rows.Scan(&sequence, &size); err != nil {
			break
		}
		total += size
		if err = reserve(metadataFloor + 4*total); err != nil {
			break
		}
		ids = append(ids, sequence)
	}
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil || len(ids) == 0 {
		return nil, metadataFloor, err
	}
	// Pin the selected IDs. New appends cannot expand this reservation, and
	// changed envelopes fail closed before SQLite returns their body or refs.
	arguments := append(ids, total, total)
	page, err := s.recordPage(ctx, `WITH selected AS (
 SELECT *,SUM(length(body)+length(refs)) OVER (ORDER BY sequence DESC) AS charged FROM messages WHERE sequence IN (`+placeholders(len(ids))+`)
) SELECT sequence,'message',CASE WHEN charged<=? THEN body END,CASE WHEN charged<=? THEN refs END,raw_bytes FROM selected ORDER BY sequence DESC`, arguments...)
	return page, metadataFloor + 4*total, err
}
