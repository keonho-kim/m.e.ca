// Purpose: Store one current independent LAT memo without altering main compaction memory.
// Pattern: Transactional derived-state repository with integrity-checked compression.
// Usage: The runtime owner publishes a memo only after a completed background summary.
// Related: internal/agent/lat/memory/types.go, internal/storage/sqlite/lat_sources.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	json "github.com/goccy/go-json"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) LATMemory(ctx context.Context, session string) (latmemory.State, error) {
	var state latmemory.State
	var body []byte
	var codec, size int
	var digest string
	err := s.db.QueryRowContext(ctx, "SELECT revision,through_timeline,body,codec,digest,raw_bytes FROM lat_memory WHERE session_id=?", session).Scan(&state.Revision, &state.Through, &body, &codec, &digest, &size)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	body, err = s.codec.Restore(codec, size, digest, body)
	if err != nil {
		return state, err
	}
	if err = json.Unmarshal(body, &state.Text); err != nil {
		return state, err
	}
	return state, state.Check()
}

func (s *Store) InstallLATMemory(ctx context.Context, session string, expected int, next latmemory.State) error {
	if err := next.Check(); err != nil {
		return err
	}
	if next.Revision != expected+1 || next.Through == 0 {
		return fmt.Errorf("invalid independent review memory revision")
	}
	body, err := json.Marshal(next.Text)
	if err != nil {
		return err
	}
	stored, codec := body, 0
	if len(body) > content.Threshold {
		compressed, err := s.codec.Encode(body)
		if err != nil {
			return err
		}
		if len(compressed) < len(body) {
			stored, codec = compressed, 2
		}
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO lat_memory(session_id,revision,through_timeline,body,codec,digest,raw_bytes)
SELECT ?,?,?,?,?,?,? WHERE ?=0 OR EXISTS(SELECT 1 FROM lat_memory WHERE session_id=? AND revision=?)
ON CONFLICT(session_id) DO UPDATE SET revision=excluded.revision,through_timeline=excluded.through_timeline,body=excluded.body,codec=excluded.codec,digest=excluded.digest,raw_bytes=excluded.raw_bytes
WHERE lat_memory.revision=? AND lat_memory.through_timeline<excluded.through_timeline`, session, next.Revision, next.Through, stored, codec, content.ID(body), len(body), expected, session, expected, expected)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return errors.Join(fmt.Errorf("stale independent review memory; previous memo retained"), err)
	}
	return nil
}
