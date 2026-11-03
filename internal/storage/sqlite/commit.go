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

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (s *Store) Commit(ctx context.Context, session string, records []runrecord.Entry) error {
	if len(records) == 0 {
		return fmt.Errorf("journal group must not be empty")
	}
	return s.commit(ctx, session, records, 0, nil)
}

func (s *Store) CommitContext(ctx context.Context, session string, records []runrecord.Entry, cursor int64) error {
	if cursor < 1 {
		return fmt.Errorf("invalid context cursor")
	}
	return s.commit(ctx, session, records, cursor, nil)
}

type journalReceipt struct{ event, timeline int64 }

func (s *Store) commit(ctx context.Context, session string, records []runrecord.Entry, cursor int64, receipt *journalReceipt) error {
	if receipt == nil {
		receipt = &journalReceipt{}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var firstUser *string
	hasMessage := false
	for _, record := range records {
		switch {
		case record.Message != nil && record.Event == nil:
			err = s.writeMessage(ctx, tx, session, *record.Message, now, receipt)
			hasMessage = true
			if firstUser == nil && record.Message.Role == "user" {
				summary := conversationSummary(record.Message.Content)
				firstUser = &summary
			}
		case record.Event != nil && record.Message == nil:
			err = s.writeEvent(ctx, tx, session, *record.Event, now, receipt)
		default:
			err = fmt.Errorf("journal record requires exactly one payload")
		}
		if err != nil {
			return errors.Join(err, tx.Rollback())
		}
	}
	if hasMessage {
		if _, err := tx.ExecContext(ctx, "UPDATE sessions SET updated_at=?,summary=COALESCE(summary,?) WHERE id=?", now, firstUser, session); err != nil {
			return errors.Join(err, tx.Rollback())
		}
	}
	if cursor > 0 {
		if _, err := tx.ExecContext(ctx, "UPDATE sessions SET context_cursor=MAX(context_cursor,?) WHERE id=?", cursor, session); err != nil {
			return errors.Join(err, tx.Rollback())
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for old := s.latestEvent.Load(); receipt.event > old; old = s.latestEvent.Load() {
		if s.latestEvent.CompareAndSwap(old, receipt.event) {
			break
		}
	}
	return nil
}
