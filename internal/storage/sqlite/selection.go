// Purpose: Persist provider selection and list conversations for explicit resume.
// Pattern: Repository.
// Usage: Called by startup resume and terminal model selection.
// Related: internal/storage/sqlite/session.go, internal/provider/connection/connection.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
)

type Selection struct {
	Connection        modelconn.Connection `json:"connection"`
	Model             string               `json:"model"`
	Thinking          *bool                `json:"thinking,omitempty"`
	ThinkingSupported bool                 `json:"thinking_supported"`
}
type Conversation struct {
	Session
	Summary, Updated string
}

func (s *Store) Conversations(ctx context.Context, offset, limit int) (out []Conversation, err error) {
	if offset < 0 || limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid conversation page")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT s.id,s.project_root,s.display_root,s.updated_at,COALESCE((SELECT json_extract(body,'$.content') FROM messages WHERE session_id=s.id AND json_extract(body,'$.role')='user' ORDER BY sequence LIMIT 1),'New conversation') FROM sessions s ORDER BY updated_at DESC,rowid DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var c Conversation
		if err = rows.Scan(&c.ID, &c.ProjectRoot, &c.DisplayRoot, &c.Updated, &c.Summary); err != nil {
			return nil, err
		}
		if len(c.Summary) > 200 {
			c.Summary = string([]rune(c.Summary)[:min(80, len([]rune(c.Summary)))])
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (s *Store) SaveSelection(ctx context.Context, session string, selection Selection) error {
	parsed, err := modelconn.New(selection.Connection.Kind, selection.Connection.Endpoint)
	if err != nil {
		return err
	}
	if parsed.ID != selection.Connection.ID || selection.Model == "" || len(selection.Model) > 512 {
		return fmt.Errorf("invalid model selection")
	}
	body, err := json.Marshal(selection)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE sessions SET selection=? WHERE id=?", body, session)
	return err
}
func (s *Store) Selection(ctx context.Context, session string) (Selection, error) {
	var body []byte
	err := s.db.QueryRowContext(ctx, "SELECT selection FROM sessions WHERE id=?", session).Scan(&body)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Selection{}, err
	}
	if len(body) == 0 {
		return Selection{}, nil
	}
	var selected Selection
	err = json.Unmarshal(body, &selected)
	if err != nil {
		return selected, err
	}
	_, err = modelconn.New(selected.Connection.Kind, selected.Connection.Endpoint)
	return selected, err
}
