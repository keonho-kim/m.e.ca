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
	err := s.db.QueryRowContext(ctx, "SELECT t.sequence\n FROM timeline t JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence\n WHERE t.sequence<=? AND m.role='assistant'\n AND COALESCE(json_extract(m.body,'$.incomplete'),0)=0\n ORDER BY t.sequence DESC LIMIT 1",

		through).Scan(&completed)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	var pin sql.NullInt64
	err = s.db.QueryRowContext(ctx, "SELECT COALESCE((\n SELECT a.sequence FROM timeline a JOIN messages am ON a.kind='message' AND a.source_sequence=am.sequence\n WHERE a.sequence<t.sequence AND am.role='assistant'\n ORDER BY a.sequence DESC LIMIT 1),t.sequence)-1\n FROM timeline t JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence\n WHERE t.sequence>? AND t.sequence<=? AND m.role IN ('user','tool')\n AND (json_array_length(m.body,'$.images')>0 OR json_type(m.body,'$.images')='null') ORDER BY t.sequence LIMIT 1",

		completed, through).Scan(&pin)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if pin.Valid {
		return pin.Int64, nil
	}
	return through, nil
}
