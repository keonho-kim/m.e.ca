// Purpose: Persist validated model messages and load bounded session history.
// Pattern: Repository.
// Usage: Used by the agent generation loop and session restoration.
// Related: internal/agent/model/model.go, internal/storage/sqlite/store.go, internal/storage/sqlite/store_test.go.

package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
)

func (store *Store) AppendMessage(ctx context.Context, session string, message modelapi.Message) error {
	return store.Commit(ctx, session, []run.JournalRecord{{Message: &message}})
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

func (store *Store) Messages(ctx context.Context, session string, limit int) (messages []modelapi.Message, err error) {
	if limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("history limit must be between 1 and 1000")
	}
	page, err := store.recordPage(ctx, "SELECT sequence,'message',body,refs,raw_bytes FROM messages WHERE session_id=? ORDER BY sequence DESC LIMIT ?", session, limit)
	if err != nil {
		return nil, err
	}
	bytes := 0
	for _, record := range page {
		content, err := store.unpackRecord(ctx, record, nil)
		if err != nil {
			return nil, err
		}
		bytes += len(content)
		if bytes > maxEventBytes {
			break
		}
		var message modelapi.Message
		if err := json.Unmarshal(content, &message); err != nil {
			return nil, fmt.Errorf("invalid stored message: %w", err)
		}
		if err := message.Check(); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, nil
}
