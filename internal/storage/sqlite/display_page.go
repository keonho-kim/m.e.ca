// Purpose: Read visible history envelopes and public arguments before payload hydration.
// Pattern: SQLite read projection.
// Usage: Timeline closes this page query before resolving selected payload references.
// Related: internal/storage/sqlite/timeline.go, internal/storage/history/display.go.
package sqlite

import (
	"context"
	"errors"
)

type displayRecord struct {
	record    storedRecord
	arguments []byte
	internal  bool
}

func (s *Store) displayPage(ctx context.Context, session string, before int64, limit int) (page []displayRecord, err error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,t.kind,COALESCE(m.body,e.body),COALESCE(m.refs,e.refs),COALESCE(m.raw_bytes,e.raw_bytes),COALESCE(e.display_args,'{}'),COALESCE(json_extract(e.body,'$.internal'),0)
FROM timeline t LEFT JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence
LEFT JOIN tool_events e ON t.kind='event' AND t.source_sequence=e.sequence
WHERE t.session_id=? AND t.sequence<? AND (
 (m.role IN ('user','assistant') AND m.content_bytes>0) OR
 (e.kind IN ('thinking','progress','lat_clean','lat_feedback','lat_unavailable','lat_discarded','lat_memory_unavailable','tool_succeeded','tool_failed','tool_interrupted','denied','command_finished','run_checkpoint') AND COALESCE(json_extract(e.body,'$.internal'),0)=0) OR
 (COALESCE(json_extract(e.body,'$.internal'),0)=1 AND json_type(e.body,'$.lat')='object' AND (
  e.kind IN ('thinking','tool_succeeded','tool_failed','tool_interrupted','denied','command_finished') OR
  (e.kind='lat_message' AND json_extract(e.body,'$.message.role')='assistant'
   AND (json_extract(e.body,'$.message.content')<>'' OR json_type(e.body,'$.message.content')='null')
   AND NOT (json_extract(e.body,'$.lat.phase')='review' AND COALESCE(json_array_length(e.body,'$.message.calls'),0)=0
    AND COALESCE(json_extract(e.body,'$.message.incomplete'),0)=0)))))
ORDER BY t.sequence DESC LIMIT ?`, session, before, limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var entry displayRecord
		r := &entry.record
		if err = rows.Scan(&r.sequence, &r.kind, &r.body, &r.refs, &r.size, &entry.arguments, &entry.internal); err != nil {
			return nil, err
		}
		page = append(page, entry)
	}
	return page, rows.Err()
}
