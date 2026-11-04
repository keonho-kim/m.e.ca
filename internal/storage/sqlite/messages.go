// Purpose: Persist validated model messages and their ordered timeline references.
// Pattern: Repository.
// Usage: The journal commits messages before execution or after observed results.
// Related: internal/agent/model/model.go, internal/storage/sqlite/store.go, internal/storage/sqlite/store_test.go.

package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (store *Store) AppendMessage(ctx context.Context, session string, message modelapi.Message) error {
	return store.Commit(ctx, session, []runrecord.Entry{{Message: &message}})
}

func (store *Store) writeMessage(ctx context.Context, transaction *sql.Tx, session string, message modelapi.Message, now string, receipt *journalReceipt) error {
	if err := message.Check(); err != nil {
		return err
	}
	if message.ID == "" {
		message.ID = "message_" + rand.Text()
	}
	packed, refs, size, err := store.packMessage(ctx, transaction, message)
	if err != nil {
		return err
	}
	inserted, err := transaction.ExecContext(ctx, "INSERT INTO messages(session_id,body,refs,raw_bytes,role,content_bytes,created_at) VALUES(?,?,?,?,?,?,?)", session, packed, refs, size, message.Role, len(message.Content), now)
	if err != nil {
		return err
	}
	sequence, err := inserted.LastInsertId()
	if err != nil {
		return err
	}
	timeline, err := transaction.ExecContext(ctx, "INSERT INTO timeline(session_id,kind,source_sequence) VALUES(?,'message',?)", session, sequence)
	if err != nil {
		return err
	}
	receipt.timeline, err = timeline.LastInsertId()
	if err != nil {
		return err
	}
	return nil
}
