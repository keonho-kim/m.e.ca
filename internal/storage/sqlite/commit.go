// Purpose: Commit ordered message/event groups at one explicit durability boundary.
// Pattern: SQLite transaction adapter.
// Usage: Runtime groups records before effects or after observations; single-write APIs use the same path.
// Related: internal/storage/sqlite/messages.go, internal/storage/sqlite/events.go, internal/agent/run/journal.go.
package sqlite

import (
	"context"
	"errors"

	"time"

	"github.com/keonho-kim/meca/internal/agent/team/channel"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) Commit(ctx context.Context, session string, records []runrecord.Entry) error {
	if err := s.checkSession(session); err != nil {
		return err
	}
	if len(records) == 0 {
		return faults.Errorf(diagcatalog.SqliteJournalGroupMustNotBeEmpty)
	}
	return s.commit(ctx, session, records, 0, nil)
}

func (s *Store) CommitContext(ctx context.Context, session string, records []runrecord.Entry, cursor int64) error {
	if err := s.checkSession(session); err != nil {
		return err
	}
	if cursor < 1 {
		return faults.Errorf(diagcatalog.SqliteInvalidContextCursor)
	}
	return s.commit(ctx, session, records, cursor, nil)
}

type journalReceipt struct {
	event, timeline int64
	public          *channel.Message
}

func (s *Store) commit(ctx context.Context, session string, records []runrecord.Entry, cursor int64, receipt *journalReceipt) (resultErr error) {
	started := time.Now()
	defer func() {
		elapsed := time.Since(started).Milliseconds()
		if resultErr != nil {
			diagnostics.Log(ctx, diagcatalog.StorageCommitFailed, "component", "storage", "session_id", session, "duration_ms", elapsed, "error_code", diagnostics.Code(resultErr))
		} else if elapsed >= 100 {
			diagnostics.Log(ctx, diagcatalog.StorageCommitSlow, "component", "storage", "session_id", session, "duration_ms", elapsed)
		} else {
			diagnostics.Log(ctx, diagcatalog.StorageCommitted, "component", "storage", "session_id", session, "duration_ms", elapsed, "count", len(records))
		}
	}()
	if err := checkJournal(ctx, records); err != nil {
		return err
	}
	release, err := reserveJournal(ctx, records)
	if err != nil {
		return err
	}
	defer release()
	prepared, err := s.prepareJournal(ctx, records)
	if err != nil {
		return err
	}
	if receipt == nil {
		receipt = &journalReceipt{}
	}
	waitStarted := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if waited := time.Since(waitStarted).Milliseconds(); waited >= 100 {
		diagnostics.Log(ctx, diagcatalog.StorageWriterWait, "component", "storage", "session_id", session, "duration_ms", waited)
	}
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var firstUser *string
	hasMessage := false
	for index, record := range records {
		var value preparedRecord
		if index < len(prepared) {
			value = prepared[index]
			prepared[index] = preparedRecord{}
		} else {
			encoder := journalEncoder{store: s, ctx: ctx, lookup: tx}
			value, err = encoder.prepare(record)
			if err != nil {
				return errors.Join(err, tx.Rollback())
			}
		}
		err = value.write(ctx, s, tx, session, now, receipt)
		if record.Message != nil {
			hasMessage = true
			if firstUser == nil && record.Message.Role == "user" {
				summary := conversationSummary(record.Message.Content)
				firstUser = &summary
			}
		}
		if err != nil {
			return errors.Join(err, tx.Rollback())
		}
	}
	if receipt.public != nil {
		if err := s.writePublicSnapshot(ctx, tx, receipt.public, receipt.timeline); err != nil {
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
		return faults.New(diagcatalog.PersistenceFailed, diagcatalog.WrappedCause, err)
	}
	for old := s.latestEvent.Load(); receipt.event > old; old = s.latestEvent.Load() {
		if s.latestEvent.CompareAndSwap(old, receipt.event) {
			break
		}
	}
	return nil
}
