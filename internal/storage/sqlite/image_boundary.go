// Purpose: Protect unread image tool groups until a completed assistant generation follows them.
// Pattern: Metadata-only persistence projection.
// Usage: CompactionBoundary caps its selected prefix before an unconsumed image group.
// Related: internal/storage/sqlite/memory_boundary.go, internal/storage/sqlite/messages.go.
package sqlite

import (
	"context"
	"database/sql"
)

func (s *Store) unreadImageBoundary(ctx context.Context, session string, through int64) (int64, error) {
	// A completed generation consumes every earlier image. Reverse lookup stops
	// at the latest such message rather than revisiting all historical images.
	var completed int64
	err := s.db.QueryRowContext(ctx, `SELECT t.sequence
 FROM timeline t JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence
 WHERE t.session_id=? AND t.sequence<=? AND m.role='assistant'
 AND COALESCE(json_extract(m.body,'$.incomplete'),0)=0
 ORDER BY t.sequence DESC LIMIT 1`, session, through).Scan(&completed)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	var pin sql.NullInt64
	err = s.db.QueryRowContext(ctx, `SELECT COALESCE((
 SELECT a.sequence FROM timeline a JOIN messages am ON a.kind='message' AND a.source_sequence=am.sequence
 WHERE a.session_id=t.session_id AND a.sequence<t.sequence AND am.role='assistant'
 ORDER BY a.sequence DESC LIMIT 1),t.sequence)-1
 FROM timeline t JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence
 WHERE t.session_id=? AND t.sequence>? AND t.sequence<=? AND m.role='tool'
 AND json_array_length(m.body,'$.images')>0 ORDER BY t.sequence LIMIT 1`, session, completed, through).Scan(&pin)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if pin.Valid {
		return pin.Int64, nil
	}
	return through, nil
}
