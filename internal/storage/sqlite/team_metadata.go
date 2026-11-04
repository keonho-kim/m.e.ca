// Purpose: Resolve public record identity and size without loading referenced bodies.
// Pattern: Narrow SQLite projection.
// Usage: Reply validation and compaction references query the owning public session.
// Related: internal/agent/team/channel/message.go, internal/storage/sqlite/team_messages.go.
package sqlite

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func (s *Store) TeamMessageMetadata(ctx context.Context, sequence int64) (channel.Metadata, error) {
	value := channel.Metadata{Sequence: sequence}
	err := s.db.QueryRowContext(ctx, `SELECT CASE WHEN t.kind='message' THEN 'user' ELSE json_extract(e.body,'$.agent') END,
 CASE WHEN t.kind='message' THEN 'user_message' ELSE e.kind END,COALESCE(m.raw_bytes,e.raw_bytes)
 FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence
 LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence
 WHERE t.session_id=? AND t.sequence=? AND (m.role='user' OR e.kind IN
 ('team_message','team_progress','team_work','team_discussion','team_observation','team_status'))`, s.session.ID, sequence).Scan(&value.From, &value.Kind, &value.Bytes)
	return value, err
}
