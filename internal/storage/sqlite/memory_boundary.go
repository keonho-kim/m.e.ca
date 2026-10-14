// Purpose: Choose an archived prefix while retaining recent whole messages verbatim.
// Pattern: Repository projection.
// Usage: Compaction asks for a byte-budgeted recent tail before generating summaries.
// Related: internal/agent/compact/controller.go, internal/storage/sqlite/context.go.
package sqlite

import (
	"context"
	"errors"
	"fmt"
)

func (s *Store) CompactionBoundary(ctx context.Context, session string, through int64, keepBytes int) (boundary int64, err error) {
	if through < 0 || keepBytes < 0 {
		return 0, fmt.Errorf("invalid compaction tail bounds")
	}
	if keepBytes == 0 {
		return through, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,length(CAST(m.body AS BLOB)) FROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence<=? ORDER BY t.sequence DESC`, session, through)
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	used := 0
	for rows.Next() {
		var sequence int64
		var size int
		if err := rows.Scan(&sequence, &size); err != nil {
			return 0, err
		}
		used += size
		if used >= keepBytes {
			return sequence - 1, nil
		}
	}
	return 0, rows.Err()
}
