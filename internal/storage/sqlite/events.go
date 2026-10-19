// Purpose: Persist ordered runtime events, including final per-call token usage.
// Pattern: Repository.
// Usage: Implements the agent task journal contract and feeds restored terminal history.
// Related: internal/agent/run/run.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"time"
)

// Bound original captures and nested execution records independently of context projections.
const maxEventBytes = 512 * 1024 * 1024

func (s *Store) RecordEvent(ctx context.Context, session string, event run.Event) error {
	if err := event.Check(); err != nil {
		return err
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if len(body) > maxEventBytes {
		return fmt.Errorf("event exceeds limit")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	inserted, err := tx.ExecContext(ctx, "INSERT INTO tool_events(session_id,body,created_at) VALUES(?,?,?)", session, body, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	sequence, err := inserted.LastInsertId()
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO timeline(session_id,kind,source_sequence) VALUES(?,'event',?)", session, sequence); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}
