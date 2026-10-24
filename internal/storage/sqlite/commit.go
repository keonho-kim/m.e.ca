// Purpose: Commit ordered message/event groups at one explicit durability boundary.
// Pattern: SQLite transaction adapter.
// Usage: Runtime groups records before effects or after observations; single-write APIs use the same path.
// Related: internal/storage/sqlite/messages.go, internal/storage/sqlite/events.go, internal/agent/run/journal.go.
package sqlite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/keonho-kim/meca/internal/agent/run"
)

func (s *Store) Commit(ctx context.Context, session string, records []run.JournalRecord) error {
	if len(records) == 0 {
		return fmt.Errorf("journal group must not be empty")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, record := range records {
		switch {
		case record.Message != nil && record.Event == nil:
			err = writeMessage(ctx, tx, session, *record.Message, now)
		case record.Event != nil && record.Message == nil:
			err = writeEvent(ctx, tx, session, *record.Event, now)
		default:
			err = fmt.Errorf("journal record requires exactly one payload")
		}
		if err != nil {
			return errors.Join(err, tx.Rollback())
		}
	}
	return tx.Commit()
}
