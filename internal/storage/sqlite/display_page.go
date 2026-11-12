// Purpose: Read visible history envelopes and public arguments before payload hydration.
// Pattern: SQLite read projection.
// Usage: Timeline closes this page query before resolving selected payload references.
// Related: internal/storage/sqlite/timeline.go, internal/storage/history/display.go.
package sqlite

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"
)

const visibleHistoryJoin = `FROM timeline t LEFT JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence
LEFT JOIN tool_events e ON t.kind='event' AND t.source_sequence=e.sequence`
const visibleHistoryPredicate = `(m.role IN ('user','assistant') AND (m.content_bytes>0 OR json_array_length(m.body,'$.images')>0 OR json_type(m.body,'$.images')='null')) OR
 e.kind IN ('thinking','progress','skill_loaded','tool_succeeded','tool_failed','tool_interrupted','denied','command_finished','run_checkpoint','team_message','team_progress','team_work','team_discussion','team_notice','team_reference')`

type displayRecord struct {
	eventKind string
	source    int64
	record    storedRecord
	arguments []byte
}

func (s *Store) displayPage(ctx context.Context, session string, before int64, limit int, maxBytes int) (page []displayRecord, err error) {
	limited := maxBytes > 0
	retained := 0
	if maxBytes == 0 {
		maxBytes = content.MaxBytes
	}
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,t.kind,CASE WHEN length(COALESCE(m.body,e.body))<=? THEN COALESCE(m.body,e.body) END,COALESCE(m.refs,e.refs),COALESCE(m.raw_bytes,e.raw_bytes),COALESCE(e.display_args,'{}'),t.source_sequence,COALESCE(e.kind,'')
`+visibleHistoryJoin+` WHERE t.sequence<? AND (`+visibleHistoryPredicate+`) ORDER BY t.sequence DESC LIMIT ?`,

		maxBytes, before, limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var entry displayRecord
		r := &entry.record
		if err = rows.Scan(&r.sequence, &r.kind, &r.body, &r.refs, &r.size, &entry.arguments, &entry.source, &entry.eventKind); err != nil {
			return nil, err
		}
		if r.body == nil {
			if len(page) > 0 {
				break
			}
			return nil, history.ErrReadLimit
		}
		size := len(r.body) + len(r.refs) + len(entry.arguments)
		if limited && size > maxBytes-retained {
			if len(page) > 0 {
				break
			}
			return nil, history.ErrReadLimit
		}
		retained += size
		page = append(page, entry)
	}
	return page, rows.Err()
}
