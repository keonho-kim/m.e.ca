// Purpose: Load every uncompacted model message in chronological order without a newest-record cutoff.
// Pattern: Repository projection.
// Usage: The agent journal supplies a committed compaction timeline boundary and a byte budget.
// Related: internal/storage/sqlite/messages.go, internal/storage/sqlite/timeline.go, internal/agent/run/context.go.
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func (s *Store) ContextMessages(ctx context.Context, session string, after int64, maxBytes int) (messages []modelapi.Message, err error) {
	if after < 0 || maxBytes < 1 {
		return nil, fmt.Errorf("invalid conversation context bounds")
	}
	page, err := s.recordPage(ctx, `SELECT t.sequence,t.kind,m.body,m.refs,m.raw_bytes FROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence>? ORDER BY t.sequence`, session, after)
	if err != nil {
		return nil, err
	}
	bytes := 0
	for _, record := range page {
		bytes += record.size
		if bytes > maxBytes {
			return nil, fmt.Errorf("uncompacted conversation exceeds %d bytes; no messages were omitted", maxBytes)
		}
		body, err := s.unpackRecord(ctx, record, nil)
		if err != nil {
			return nil, err
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
	return messages, nil
}
