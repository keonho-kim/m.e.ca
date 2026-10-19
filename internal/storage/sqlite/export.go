// Purpose: Stream an original session timeline as ordered JSONL without loading the whole history.
// Pattern: Persistence export adapter.
// Usage: v --session ID --export writes this stream to stdout.
// Related: internal/storage/sqlite/timeline.go, cmd/v/headless.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func (s *Store) Export(ctx context.Context, session string, output io.Writer) (err error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,t.kind,CASE t.kind WHEN 'message' THEN m.body ELSE e.body END FROM timeline t LEFT JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence LEFT JOIN tool_events e ON t.kind='event' AND t.source_sequence=e.sequence WHERE t.session_id=? ORDER BY t.sequence`, session)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	encoder := json.NewEncoder(output)
	for rows.Next() {
		var record struct {
			Sequence int64           `json:"sequence"`
			Kind     string          `json:"kind"`
			Body     json.RawMessage `json:"body"`
		}
		if err = rows.Scan(&record.Sequence, &record.Kind, &record.Body); err != nil {
			return err
		}
		if len(record.Body) > maxEventBytes || !json.Valid(record.Body) {
			return fmt.Errorf("invalid session export record")
		}
		if err = encoder.Encode(record); err != nil {
			return err
		}
	}
	return rows.Err()
}
