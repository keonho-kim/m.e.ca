// Purpose: Prepare model messages and atomically persist their ordered timeline references.
// Pattern: SQLite journal adapter.
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

func (p *journalEncoder) message(value *preparedRecord) error {
	message := *value.source.Message
	if message.ID == "" {
		message.ID = "message_" + rand.Text()
	}
	packed, refs, size, err := p.packMessage(message)
	if err != nil {
		return err
	}
	value.body, value.referenceSlots, value.size = packed, refs, size
	return nil
}

func (store *Store) writeMessage(ctx context.Context, transaction *sql.Tx, session string, value *preparedRecord, now string, receipt *journalReceipt) error {
	message := value.source.Message
	inserted, err := transaction.ExecContext(ctx, "INSERT INTO messages(body,refs,raw_bytes,role,content_bytes,created_at) VALUES(?,?,?,?,?,?)", value.body, value.refs, value.size, message.Role, len(message.Content), now)
	if err != nil {
		return err
	}
	sequence, err := inserted.LastInsertId()
	if err != nil {
		return err
	}
	timeline, err := transaction.ExecContext(ctx, "INSERT INTO timeline(kind,source_sequence) VALUES('message',?)", sequence)
	if err != nil {
		return err
	}
	receipt.timeline, err = timeline.LastInsertId()
	if err != nil {
		return err
	}
	return nil
}
