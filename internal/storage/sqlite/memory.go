// Purpose: Replace derived context occupancy and compaction state while preserving original history.
// Pattern: Repository.
// Usage: Each session owns its compaction revisions and token boundaries.
// Related: internal/agent/compact/types.go, internal/storage/sqlite/memory_sources.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"time"

	json "github.com/goccy/go-json"
	"github.com/keonho-kim/meca/internal/agent/compact"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) AddContextTokens(ctx context.Context, session string, count compact.TokenCount) error {
	if err := s.checkSession(session); err != nil {
		return err
	}
	if count.Amount < 0 || count.Amount > 1000000000 {
		return faults.Errorf(diagcatalog.SqliteInvalidConversationTokenCount)
	}
	switch count.Kind {
	case "request":
	default:
		return faults.Errorf(diagcatalog.SqliteInvalidConversationTokenSource)
	}
	_, err := s.execWrite(ctx, "INSERT INTO conversation_tokens(singleton,amount,estimated,kind,through_timeline) SELECT 1,?,?,?,COALESCE(MAX(sequence),0) FROM timeline WHERE true ON CONFLICT(singleton) DO UPDATE SET sequence=excluded.sequence,amount=excluded.amount,estimated=excluded.estimated,kind=excluded.kind,through_timeline=excluded.through_timeline", count.Amount, count.Estimated, count.Kind)
	return err
}
func (s *Store) Memory(ctx context.Context, session string) (compact.Memory, error) {
	if err := s.checkSession(session); err != nil {
		return compact.Memory{}, err
	}
	var body []byte
	var codec, size int
	var digest string
	err := s.db.QueryRowContext(ctx, "SELECT body,codec,digest,raw_bytes FROM memories WHERE singleton=1").Scan(&body, &codec, &digest, &size)
	if errors.Is(err, sql.ErrNoRows) {
		return compact.Memory{}, nil
	}
	if err != nil {
		return compact.Memory{}, err
	}
	if len(body) > recordcontent.MaxBytes {
		return compact.Memory{}, faults.Errorf(diagcatalog.SqliteStoredMemoryExceedsLimit)
	}
	body, err = s.codec.Restore(codec, size, digest, body)
	if err != nil {
		return compact.Memory{}, err
	}
	var memory compact.Memory
	err = json.Unmarshal(body, &memory)
	if err != nil {
		return memory, err
	}
	return memory, memory.Check(compact.MaxBodyBytes)
}
func (s *Store) InstallMemory(ctx context.Context, session string, expected int, next compact.Memory) error {
	if err := s.checkSession(session); err != nil {
		return err
	}
	if next.Revision != expected+1 {
		return faults.Errorf(diagcatalog.SqliteInvalidCompactionRevision)
	}
	if err := next.Check(compact.MaxBodyBytes); err != nil {
		return err
	}
	body, err := json.Marshal(next)
	if err != nil {
		return err
	}
	codec, stored := 0, body
	if len(body) > recordcontent.Threshold {
		encoded, err := s.codec.Encode(body)
		if err != nil {
			return err
		}
		if len(encoded) < len(body) {
			codec, stored = 2, encoded
		}
	}
	result, err := s.execWrite(ctx, `INSERT INTO memories(singleton,revision,through_timeline,through_usage,body,codec,digest,raw_bytes,created_at)
SELECT 1,?,?,?,?,?,?,?,? WHERE ?=0 OR EXISTS(SELECT 1 FROM memories WHERE singleton=1 AND revision=?)
ON CONFLICT(singleton) DO UPDATE SET revision=excluded.revision,through_timeline=excluded.through_timeline,through_usage=excluded.through_usage,body=excluded.body,codec=excluded.codec,digest=excluded.digest,raw_bytes=excluded.raw_bytes,created_at=excluded.created_at
WHERE memories.revision=? AND memories.through_timeline<excluded.through_timeline AND memories.through_usage<=excluded.through_usage`, next.Revision, next.Through, next.UsageThrough, stored, codec, recordcontent.ID(body), len(body), time.Now().UTC().Format(time.RFC3339Nano), expected, expected, expected)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return errors.Join(faults.Errorf(diagcatalog.SqliteStaleCompactionRevisionPreviousMemoryRetained), err)
	}
	return nil
}
