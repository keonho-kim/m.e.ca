// Purpose: Prepare and persist ordered runtime events at their journal durability boundary.
// Pattern: SQLite journal adapter.
// Usage: Implements the agent task journal contract and feeds restored terminal history.
// Related: internal/agent/run/run.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"context"
	"database/sql"

	json "github.com/goccy/go-json"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) RecordEvent(ctx context.Context, session string, event runrecord.Event) error {
	return s.Commit(ctx, session, []runrecord.Entry{{Event: &event}})
}

func (p *journalEncoder) event(value *preparedRecord) error {
	s := p.store
	event := *value.source.Event
	if event.Kind == "run_checkpoint" && event.Checkpoint != nil {
		copy := *event.Checkpoint
		event.Checkpoint = &copy
		var err error
		value.state, err = p.prepareRunState(event)
		if err != nil {
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
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if len(body) > recordcontent.MaxBytes {
		return faults.Errorf(diagcatalog.SqliteEventExceedsLimit)
	}
	sharedField := ""
	if event.Kind == "thinking" && len(event.Thinking) >= recordcontent.SharedThreshold {
		sharedField = "thinking"
	}
	packed, refs, err := p.pack(body, sharedField)
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
	memoryBytes := len(text)
	if ref, size, err := s.publicMemoryReference(p.ctx, event); err != nil {
		return err
	} else if ref != 0 {
		value.memorySource = ref
		role, text, memoryBytes = "peer", "", size
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
	value.body, value.referenceSlots, value.size, value.display = packed, refs, len(body), displayArgs
	value.memoryRole, value.memoryText, value.memoryBytes = role, text, memoryBytes
	return nil
}

func (s *Store) writeEvent(ctx context.Context, tx *sql.Tx, session string, value *preparedRecord, now string, receipt *journalReceipt) error {
	if value.state != nil {
		if err := s.writeRunState(ctx, tx, session, value.state); err != nil {
			return err
		}
	}
	if value.body == nil {
		return nil
	}
	inserted, err := tx.ExecContext(ctx, "INSERT INTO tool_events(body,refs,raw_bytes,kind,memory_role,memory_text,memory_bytes,memory_source,display_args,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)", value.body, value.refs, value.size, value.source.Event.Kind, value.memoryRole, value.memoryText, value.memoryBytes, value.memorySource, value.display, now)
	if err != nil {
		return err
	}
	sequence, err := inserted.LastInsertId()
	if err != nil {
		return err
	}
	timeline, err := tx.ExecContext(ctx, "INSERT INTO timeline(kind,source_sequence) VALUES('event',?)", sequence)
	if err != nil {
		return err
	}
	receipt.event = sequence
	receipt.timeline, err = timeline.LastInsertId()
	if err != nil {
		return err
	}
	return nil
}
