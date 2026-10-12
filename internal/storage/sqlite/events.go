// Purpose: Persist ordered tool and approval events and recover interrupted operations.
// Pattern: Repository.
// Usage: Implements the agent task journal contract and feeds restored terminal history.
// Related: internal/agent/run/run.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/keonho-kim/v-code/internal/agent/run"
)

func (s *Store) RecordEvent(ctx context.Context, session string, event run.Event) error {
	if event.Checkpoint != nil {
		if err := event.Checkpoint.Check(); err != nil {
			return err
		}
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if len(body) > 8*1024*1024 {
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
func (s *Store) Events(ctx context.Context, session string, before int64, limit int) (events []run.Event, cursor int64, err error) {
	if limit < 1 || limit > 1000 {
		return nil, 0, fmt.Errorf("invalid event page")
	}
	if before == 0 {
		before = 1 << 62
	}
	rows, err := s.db.QueryContext(ctx, "SELECT sequence,body FROM tool_events WHERE session_id=? AND sequence<? ORDER BY sequence DESC LIMIT ?", session, before, limit)
	if err != nil {
		return nil, 0, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var body []byte
		if err = rows.Scan(&cursor, &body); err != nil {
			return nil, 0, err
		}
		if len(body) > 8*1024*1024 {
			return nil, 0, fmt.Errorf("stored event exceeds limit")
		}
		var event run.Event
		if err = json.Unmarshal(body, &event); err != nil {
			return nil, 0, err
		}
		events = append(events, event)
	}
	for l, r := 0, len(events)-1; l < r; l, r = l+1, r-1 {
		events[l], events[r] = events[r], events[l]
	}
	return events, cursor, rows.Err()
}
