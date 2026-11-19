// Purpose: Protect unread image and browser observation groups across context compaction.
// Pattern: Metadata-only persistence projection.
// Usage: CompactionBoundary keeps the unread result and the assistant call that requested it.
// Related: internal/storage/sqlite/memory_boundary.go, internal/storage/sqlite/messages.go.
package sqlite

import (
	"context"
	"database/sql"
)

func (s *Store) unreadObservationBoundary(ctx context.Context, through, completed int64) (int64, error) {
	var first int64
	err := s.db.QueryRowContext(ctx, `
 SELECT t.sequence FROM timeline t JOIN messages m ON m.sequence=t.source_sequence
 WHERE t.kind='message' AND t.sequence>? AND t.sequence<=? AND m.role IN ('user','tool')
 AND (json_array_length(m.body,'$.images')>0 OR json_type(m.body,'$.images')='null'
      OR (`+transientObservationSQL+`))
 ORDER BY t.sequence LIMIT 1`, completed, through).Scan(&first)
	if err == sql.ErrNoRows {
		return through, nil
	}
	if err != nil {
		return 0, err
	}
	if completed > 0 {
		return completed - 1, nil
	}
	return first - 1, nil
}
