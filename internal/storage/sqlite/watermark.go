// Purpose: Read measured context, trailing observations and compaction boundaries from one SQLite snapshot.
// Pattern: Indexed repository projection.
// Usage: Compaction checks current occupancy without loading full messages or adding persisted counters.
// Related: internal/storage/sqlite/memory.go, internal/agent/compact/controller.go.
package sqlite

import (
	"context"

	"github.com/keonho-kim/meca/internal/agent/compact"
)

func (s *Store) MemoryWatermark(ctx context.Context, session string, after int64) (compact.Watermark, error) {
	var mark compact.Watermark
	var estimated bool
	var trailingBytes int64
	err := s.db.QueryRowContext(ctx, `
WITH latest AS (
 SELECT MAX(sequence) AS sequence FROM conversation_tokens
 WHERE session_id=?1 AND sequence>?2 AND kind='request'
)
SELECT COALESCE(sample.amount,0), COALESCE(sample.sequence,?2), COALESCE(sample.estimated,0),
 COALESCE((SELECT SUM(COALESCE(m.content_bytes,e.memory_bytes)) FROM timeline t
 LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence
 LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence AND e.kind='lat_feedback'
 WHERE t.session_id=?1 AND t.sequence>MAX(COALESCE(sample.through_timeline,0),
 COALESCE((SELECT through_timeline FROM memories WHERE session_id=?1 ORDER BY revision DESC LIMIT 1),0))
 AND (m.role IN ('user','tool','system') OR e.kind='lat_feedback')),0),
 COALESCE((SELECT MAX(sequence) FROM timeline WHERE session_id=?1),0)
FROM latest LEFT JOIN conversation_tokens sample ON sample.sequence=latest.sequence`, session, after).Scan(&mark.Tokens, &mark.Usage, &estimated, &trailingBytes, &mark.Timeline)
	if err != nil {
		return mark, err
	}
	mark.Tokens += compact.EstimateBytes(trailingBytes).Amount
	mark.Estimated = estimated || trailingBytes > 0
	return mark, nil
}
