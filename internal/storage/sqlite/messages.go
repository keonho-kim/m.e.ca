// Purpose: Persist validated model messages and load bounded session history.
// Pattern: Repository.
// Usage: Used by the agent generation loop and session restoration.
// Related: internal/agent/model/model.go, internal/storage/sqlite/store.go, internal/storage/sqlite/store_test.go.

package sqlite

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"time"
)

func (store *Store) AppendMessage(ctx context.Context, session string, message modelapi.Message) error {
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
	transaction, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	inserted, err := transaction.ExecContext(ctx, "INSERT INTO messages(session_id,body,created_at) VALUES(?,?,?)", session, content, now)
	if err != nil {
		return errors.Join(err, transaction.Rollback())
	}
	sequence, err := inserted.LastInsertId()
	if err != nil {
		return errors.Join(err, transaction.Rollback())
	}
	if _, err = transaction.ExecContext(ctx, "INSERT INTO timeline(session_id,kind,source_sequence) VALUES(?,'message',?)", session, sequence); err != nil {
		return errors.Join(err, transaction.Rollback())
	}
	if _, err := transaction.ExecContext(ctx, "UPDATE sessions SET updated_at=? WHERE id=?", now, session); err != nil {
		return errors.Join(err, transaction.Rollback())
	}
	return transaction.Commit()
}

func (store *Store) Messages(ctx context.Context, session string, limit int) (messages []modelapi.Message, err error) {
	if limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("history limit must be between 1 and 1000")
	}
	rows, err := store.db.QueryContext(ctx, "SELECT body FROM messages WHERE session_id=? ORDER BY sequence DESC LIMIT ?", session, limit)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	bytes := 0
	for rows.Next() {
		var content []byte
		if err := rows.Scan(&content); err != nil {
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, nil
}
