// Purpose: Persist ordered runtime events, including final per-call token usage.
// Pattern: Repository.
// Usage: Implements the agent task journal contract and feeds restored terminal history.
// Related: internal/agent/run/run.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	json "github.com/goccy/go-json"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) RecordEvent(ctx context.Context, session string, event runrecord.Event) error {
	return s.Commit(ctx, session, []runrecord.Entry{{Event: &event}})
}

func (s *Store) writeEvent(ctx context.Context, tx *sql.Tx, session string, event runrecord.Event, now string) error {
	if err := event.Check(); err != nil {
		return err
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if len(body) > recordcontent.MaxBytes {
		return fmt.Errorf("event exceeds limit")
	}
	packed, refs, err := s.packRecord(ctx, tx, body)
	if err != nil {
		return err
	}
	role, text := "", ""
	projected, err := history.Compaction([]history.Record{{Kind: "event", Event: event}})
	if err != nil {
		return err
	}
	if len(projected) > 0 {
		role, text = projected[0].Role, projected[0].Text
	}
	inserted, err := tx.ExecContext(ctx, "INSERT INTO tool_events(session_id,body,refs,raw_bytes,kind,memory_role,memory_text,created_at) VALUES(?,?,?,?,?,?,?,?)", session, packed, refs, len(body), event.Kind, role, text, now)
	if err != nil {
		return err
	}
	sequence, err := inserted.LastInsertId()
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO timeline(session_id,kind,source_sequence) VALUES(?,'event',?)", session, sequence); err != nil {
		return err
	}
	return nil
}
