// Purpose: Load every uncompacted model message in chronological order without a newest-record cutoff.
// Pattern: Repository projection.
// Usage: The agent journal supplies a committed compaction timeline boundary and a byte budget.
// Related: internal/storage/sqlite/messages.go, internal/storage/sqlite/timeline.go, internal/agent/run/context.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func (s *Store) ContextMessages(ctx context.Context, session string, after int64, maxBytes int) (messages []modelapi.Message, err error) {
	if after < 0 || maxBytes < 1 {
		return nil, fmt.Errorf("invalid conversation context bounds")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT m.body FROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence>? ORDER BY t.sequence`, session, after)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	bytes := 0
	for rows.Next() {
		var body []byte
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		bytes += len(body)
		if bytes > maxBytes {
			return nil, fmt.Errorf("uncompacted conversation exceeds %d bytes; no messages were omitted", maxBytes)
		}
		var message modelapi.Message
		if err := json.Unmarshal(body, &message); err != nil {
			return nil, err
		}
		if err := message.Check(); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}
