// Purpose: Replace one recoverable state per run without retaining intermediate snapshots in history.
// Pattern: Transactional state repository with bounded compressed values.
// Usage: Checkpoint writes update run_states; recovery and exports hydrate the canonical state.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/runs.go, internal/storage/sqlite/export.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/hex"

	json "github.com/goccy/go-json"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
)

type preparedState struct {
	storedValue
	run string
}

// Only the immutable objective is shared. Mutable file facts stay in the one
// replaceable run row so updates cannot accumulate obsolete payload snapshots.
type stateBody struct {
	runrecord.Event
	ObjectivePayload string `json:"objective_payload,omitempty"`
}

func (p *journalEncoder) prepareRunState(event runrecord.Event) (*preparedState, error) {
	wire := stateBody{Event: event}
	if len(event.Checkpoint.Objective) > content.Threshold {
		copy := *event.Checkpoint
		id, err := p.objectivePayload(copy.ID, copy.Objective)
		if err != nil {
			return nil, err
		}
		copy.Objective = ""
		wire.Checkpoint, wire.ObjectivePayload = &copy, id
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	if len(body) > content.MaxBytes {
		return nil, faults.Errorf(diagcatalog.SqliteRunStateExceedsLimit)
	}
	value, err := p.store.encodeValue(content.ID(body), body)
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
	var wire stateBody
	if err = json.Unmarshal(raw, &wire); err != nil {
		return wire.Event, err
	}
	if wire.ObjectivePayload != "" {
		physical, quote := content.Reference(wire.ObjectivePayload)
		digest, err := hex.DecodeString(physical)
		if err != nil || len(digest) != 32 || wire.Checkpoint == nil || wire.Checkpoint.Objective != "" {
			return wire.Event, faults.Errorf(diagcatalog.ContentStoredPayloadIntegrityCheckFailed)
		}
		var codec, size int
		var body []byte
		if err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,body FROM payloads WHERE digest=?", digest).Scan(&codec, &size, &body); err != nil {
			return wire.Event, err
		}
		raw, err := s.restorePayload(ctx, codec, size, digest, body)
		if err != nil {
			return wire.Event, err
		}
		if quote {
			wire.Checkpoint.Objective = string(raw)
		} else if err := json.Unmarshal(raw, &wire.Checkpoint.Objective); err != nil {
			return wire.Event, err
		}
	}
	return wire.Event, wire.Event.Check()
}
