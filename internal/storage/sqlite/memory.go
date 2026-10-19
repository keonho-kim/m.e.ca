// Purpose: Persist context occupancy samples and compaction revisions without deleting original history.
// Pattern: Repository.
// Usage: Each session owns its compaction revisions and token boundaries.
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
	var sourceThrough int64
	err := s.db.QueryRowContext(ctx, "SELECT amount,sequence,estimated,through_timeline FROM conversation_tokens WHERE session_id=? AND sequence>? AND kind='request' ORDER BY sequence DESC LIMIT 1", session, after).Scan(&mark.Tokens, &mark.Usage, &estimated, &sourceThrough)
	if errors.Is(err, sql.ErrNoRows) {
		mark.Usage = after
	} else if err != nil {
		return mark, err
	}
	var memoryThrough int64
	if err = s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(through_timeline),0) FROM memories WHERE session_id=?", session).Scan(&memoryThrough); err != nil {
		return mark, err
	}
	sourceThrough = max(sourceThrough, memoryThrough)
	var trailingBytes int64
	err = s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(length(CAST(json_extract(m.body,'$.content') AS BLOB))),0) FROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence>? AND json_extract(m.body,'$.role') IN ('user','tool','system')`, session, sourceThrough).Scan(&trailingBytes)
	if err != nil {
		return mark, err
	}
	mark.Tokens += compact.EstimateBytes(trailingBytes).Amount
	mark.Estimated = estimated || trailingBytes > 0
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
