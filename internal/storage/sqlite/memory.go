// Purpose: Persist cumulative conversation tokens and immutable compaction revisions without deleting original history.
// Pattern: Repository.
// Usage: Implements the compaction repository used by the single session worker.
// Related: internal/agent/compact/types.go, internal/storage/sqlite/memory_sources.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"time"
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
	_, err := s.db.ExecContext(ctx, "INSERT INTO conversation_tokens(session_id,amount,estimated,kind) VALUES(?,?,?,?)", session, count.Amount, count.Estimated, count.Kind)
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
	if len(body) > maxEventBytes {
		return compact.Memory{}, fmt.Errorf("stored memory exceeds limit")
	}
	var memory compact.Memory
	err = json.Unmarshal(body, &memory)
	if err != nil {
		return memory, err
	}
	return memory, memory.Check(compact.MaxBodyBytes)
}
func (s *Store) MemoryWatermark(ctx context.Context, session string, after int64) (compact.Watermark, error) {
	var mark compact.Watermark
	var estimated bool
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(amount),0),COALESCE(MAX(sequence),?),COALESCE(MAX(estimated),0) FROM conversation_tokens WHERE session_id=? AND sequence>? AND kind='request'", after, session, after).Scan(&mark.Tokens, &mark.Usage, &estimated)
	if err != nil {
		return mark, err
	}
	mark.Estimated = estimated
	err = s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(sequence),0) FROM timeline WHERE session_id=?", session).Scan(&mark.Timeline)
	return mark, err
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
