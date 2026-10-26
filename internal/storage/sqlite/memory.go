// Purpose: Persist context occupancy samples and compaction revisions without deleting original history.
// Pattern: Repository.
// Usage: Each session owns its compaction revisions and token boundaries.
// Related: internal/agent/compact/types.go, internal/storage/sqlite/memory_sources.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	json "github.com/goccy/go-json"
	"github.com/keonho-kim/meca/internal/agent/compact"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) AddContextTokens(ctx context.Context, session string, count compact.TokenCount) error {
	if count.Amount < 0 || count.Amount > 1000000000 {
		return fmt.Errorf("invalid conversation token count")
	}
	switch count.Kind {
	case "request":
	default:
		return fmt.Errorf("invalid conversation token source")
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO conversation_tokens(session_id,amount,estimated,kind,through_timeline) SELECT ?,?,?,?,COALESCE(MAX(sequence),0) FROM timeline WHERE session_id=?", session, count.Amount, count.Estimated, count.Kind, session)
	return err
}
func (s *Store) Memory(ctx context.Context, session string) (compact.Memory, error) {
	var body []byte
	err := s.db.QueryRowContext(ctx, "SELECT body FROM memories WHERE session_id=? ORDER BY revision DESC LIMIT 1", session).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return compact.Memory{}, nil
	}
	if err != nil {
		return compact.Memory{}, err
	}
	if len(body) > recordcontent.MaxBytes {
		return compact.Memory{}, fmt.Errorf("stored memory exceeds limit")
	}
	var memory compact.Memory
	err = json.Unmarshal(body, &memory)
	if err != nil {
		return memory, err
	}
	return memory, memory.Check(compact.MaxBodyBytes)
}
func (s *Store) InstallMemory(ctx context.Context, session string, expected int, next compact.Memory) error {
	if next.Revision != expected+1 {
		return fmt.Errorf("invalid compaction revision")
	}
	if err := next.Check(compact.MaxBodyBytes); err != nil {
		return err
	}
	body, err := json.Marshal(next)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var current int
	err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(revision),0) FROM memories WHERE session_id=?", session).Scan(&current)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if current != expected {
		return errors.Join(fmt.Errorf("stale compaction revision; previous memory retained"), tx.Rollback())
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO memories(session_id,revision,through_timeline,through_usage,body,created_at) VALUES(?,?,?,?,?,?)", session, next.Revision, next.Through, next.UsageThrough, body, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}
