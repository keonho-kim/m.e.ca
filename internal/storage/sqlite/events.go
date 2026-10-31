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
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) RecordEvent(ctx context.Context, session string, event runrecord.Event) error {
	return s.Commit(ctx, session, []runrecord.Entry{{Event: &event}})
}

func (s *Store) writeEvent(ctx context.Context, tx *sql.Tx, session string, event runrecord.Event, now string) error {
	if event.Kind == "run_checkpoint" && event.Checkpoint != nil {
		copy := *event.Checkpoint
		event.Checkpoint = &copy
		if err := event.Check(); err != nil {
			return err
		}
		if err := s.writeRunState(ctx, tx, session, event); err != nil {
			return err
		}
		if copy.Phase != "finished" {
			return nil
		}
		// The timeline keeps final statistics/status; the run state owns full file facts.
		copy.Work.Files = nil
		copy.Work.Commands = nil
		copy.Objective = ""
	}
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
	memoryBytes, memoryField := len(text), ""
	if event.Kind == "lat_feedback" {
		text = ""
		memoryField = "text"
	}
	displayArgs := []byte("{}")
	if event.Call != nil {
		// A rejected argument can be valid JSON with an invalid field type.
		// Presentation must never prevent recording that failure for correction.
		if args, projectionErr := toolapi.DisplayArgs(event.Call.Arguments); projectionErr == nil {
			displayArgs, err = json.Marshal(args)
			if err != nil {
				return err
			}
		}
	}
	reviewBytes := 0
	if (role == "outcome" || role == "failure") && !event.Internal {
		reviewBytes = min(memoryBytes, latmemory.TargetBytes) + min(len(event.Output), 8192) + min(len(event.ChangePreview), 8192) + 256
	}
	inserted, err := tx.ExecContext(ctx, "INSERT INTO tool_events(session_id,body,refs,raw_bytes,kind,memory_role,memory_text,memory_field,memory_bytes,display_args,review_bytes,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)", session, packed, refs, len(body), event.Kind, role, text, memoryField, memoryBytes, displayArgs, reviewBytes, now)
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
