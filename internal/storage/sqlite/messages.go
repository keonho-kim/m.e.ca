// Purpose: Persist validated model messages and their ordered timeline references.
// Pattern: Repository.
// Usage: The journal commits messages before execution or after observed results.
// Related: internal/agent/model/model.go, internal/storage/sqlite/store.go, internal/storage/sqlite/store_test.go.

package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"

	json "github.com/goccy/go-json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (store *Store) AppendMessage(ctx context.Context, session string, message modelapi.Message) error {
	return store.Commit(ctx, session, []runrecord.Entry{{Message: &message}})
}

func (store *Store) writeMessage(ctx context.Context, transaction *sql.Tx, session string, message modelapi.Message, now string) error {
	if err := message.Check(); err != nil {
		return err
	}
	if message.ID == "" {
		message.ID = "message_" + rand.Text()
	}
	content, err := json.Marshal(message)
	if err != nil {
		return err
	}
	packed, refs, err := store.packRecord(ctx, transaction, content)
	if err != nil {
		return err
	}
	inserted, err := transaction.ExecContext(ctx, "INSERT INTO messages(session_id,body,refs,raw_bytes,role,content_bytes,created_at) VALUES(?,?,?,?,?,?,?)", session, packed, refs, len(content), message.Role, len(message.Content), now)
	if err != nil {
		return err
	}
	sequence, err := inserted.LastInsertId()
	if err != nil {
		return err
	}
	if _, err = transaction.ExecContext(ctx, "INSERT INTO timeline(session_id,kind,source_sequence) VALUES(?,'message',?)", session, sequence); err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, "UPDATE sessions SET updated_at=? WHERE id=?", now, session); err != nil {
		return err
	}
	if message.Role == "user" {
		if _, err := transaction.ExecContext(ctx, "UPDATE sessions SET summary=? WHERE id=? AND summary IS NULL", conversationSummary(message.Content), session); err != nil {
			return err
		}
	}
	return nil
}
