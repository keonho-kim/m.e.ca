// Purpose: Parse the single conversation identity and metadata inside a session database.
// Pattern: Repository boundary parser.
// Usage: File discovery and session opening verify metadata against the database filename.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/conversations.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	sessionstate "github.com/keonho-kim/v-code/internal/storage/session"
	"time"
	"unicode/utf8"
)

func readConversation(ctx context.Context, db *sql.DB, id string) (sessionstate.Conversation, error) {
	var c sessionstate.Conversation
	if err := checkSchema(ctx, db); err != nil {
		return c, err
	}
	err := db.QueryRowContext(ctx, `SELECT id,project_root,display_root,updated_at,
 COALESCE((SELECT substr(json_extract(body,'$.content'),1,200) FROM messages WHERE json_extract(body,'$.role')='user' ORDER BY sequence LIMIT 1),'New conversation')
 FROM sessions WHERE singleton=1`).Scan(&c.ID, &c.ProjectRoot, &c.DisplayRoot, &c.Updated, &c.Summary)
	if err != nil {
		return c, err
	}
	if !sessionstate.ValidID(id) || c.ID != id {
		return c, fmt.Errorf("session identity does not match database filename")
	}
	if err := sessionstate.CheckProject(c.ProjectRoot, c.DisplayRoot); err != nil {
		return c, err
	}
	if _, err := time.Parse(time.RFC3339Nano, c.Updated); err != nil {
		return c, fmt.Errorf("invalid session timestamp: %w", err)
	}
	if !utf8.ValidString(c.Summary) {
		return c, fmt.Errorf("invalid session summary")
	}
	c.Summary = string([]rune(c.Summary)[:min(80, len([]rune(c.Summary)))])
	return c, nil
}
