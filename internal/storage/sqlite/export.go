// Purpose: Export hydrated original records in bounded pages without retaining the whole session.
// Pattern: Persistence export adapter.
// Usage: meca --export reconstructs payloads only after each metadata page releases its DB connection.
// Related: internal/storage/sqlite/record_page.go, internal/storage/sqlite/record_codec.go.
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
	"io"
)

func (s *Store) Export(ctx context.Context, session string, output io.Writer) error {
	encoder := json.NewEncoder(output)
	var after int64
	for {
		page, err := s.recordPage(ctx, `SELECT t.sequence,t.kind,COALESCE(m.body,e.body),COALESCE(m.refs,e.refs),COALESCE(m.raw_bytes,e.raw_bytes) FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence>? ORDER BY t.sequence LIMIT 64`, session, after)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		for _, stored := range page {
			body, err := s.unpackRecord(ctx, stored, nil)
			if err != nil {
				return err
			}
			if len(body) > recordcontent.MaxBytes || !json.Valid(body) {
				return fmt.Errorf("invalid session export record")
			}
			record := struct {
				Sequence int64           `json:"sequence"`
				Kind     string          `json:"kind"`
				Body     json.RawMessage `json:"body"`
			}{stored.sequence, stored.kind, body}
			if err := encoder.Encode(record); err != nil {
				return err
			}
			after = stored.sequence
		}
	}
}
