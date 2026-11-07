// Purpose: Replace one recoverable state per run without retaining intermediate snapshots in history.
// Pattern: Transactional state repository with bounded compressed values.
// Usage: Checkpoint writes update run_states; recovery and exports hydrate the canonical state.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/runs.go, internal/storage/sqlite/export.go.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	json "github.com/goccy/go-json"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/content"
)

type preparedState struct {
	storedValue
	run string
}

func (s *Store) prepareRunState(event runrecord.Event) (*preparedState, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	if len(body) > content.MaxBytes {
		return nil, fmt.Errorf("run state exceeds limit")
	}
	value, err := s.encodeValue(content.ID(body), body)
	if err != nil {
		return nil, err
	}
	return &preparedState{storedValue: value, run: event.Checkpoint.ID}, nil
}

func (s *Store) writeRunState(ctx context.Context, tx *sql.Tx, session string, value *preparedState) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO run_states(run_id,body,codec,digest,raw_bytes) VALUES(?,?,?,?,?)\nON CONFLICT(run_id) DO UPDATE SET sequence=excluded.sequence,body=excluded.body,codec=excluded.codec,digest=excluded.digest,raw_bytes=excluded.raw_bytes",
		value.run, value.body, value.codec, value.id, value.size)
	return err
}

func (s *Store) runState(ctx context.Context, session, id string) (runrecord.Event, error) {
	query := `SELECT codec,digest,raw_bytes,body FROM run_states ORDER BY sequence DESC LIMIT 1`
	var args []any
	if id != "" {
		query = `SELECT codec,digest,raw_bytes,body FROM run_states WHERE run_id=?`
		args = append(args, id)
	}
	var codec, size int
	var digest string
	var body []byte
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&codec, &digest, &size, &body); err != nil {
		return runrecord.Event{}, err
	}
	raw, err := s.codec.Restore(codec, size, digest, body)
	if err != nil {
		return runrecord.Event{}, err
	}
	var event runrecord.Event
	if err = json.Unmarshal(raw, &event); err != nil {
		return event, err
	}
	return event, event.Check()
}
