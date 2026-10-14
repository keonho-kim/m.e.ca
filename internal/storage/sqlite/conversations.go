// Purpose: Discover and page session summaries directly from database files.
// Pattern: Filesystem discovery with bounded metadata projection.
// Usage: Resume and restore-last query Sessions; each read-only database handle closes immediately.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/session.go, internal/tui/startup/resume.go.
package sqlite

import (
	"context"
	"errors"
	"fmt"
	sessionstate "github.com/keonho-kim/v-code/internal/storage/session"
	"os"
	"sort"
	"time"
)

func (s *Sessions) Conversations(ctx context.Context, offset, limit int) ([]sessionstate.Conversation, error) {
	return s.conversations(ctx, "", offset, limit)
}
func (s *Sessions) conversations(ctx context.Context, project string, offset, limit int) (out []sessionstate.Conversation, err error) {
	if offset < 0 || offset > 100000 || limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid conversation page")
	}
	type entry struct {
		conversation sessionstate.Conversation
		updated      time.Time
	}
	var recent []entry
	newer := func(i, j int) bool {
		if recent[i].updated.Equal(recent[j].updated) {
			return recent[i].conversation.ID > recent[j].conversation.ID
		}
		return recent[i].updated.After(recent[j].updated)
	}
	err = s.walkDatabases(ctx, project, func(path, id, group string) error {
		db, err := connect(ctx, path, true)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		c, readErr := readConversation(ctx, db, id)
		if err := errors.Join(readErr, db.Close()); err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		key, err := sessionstate.ProjectKey(c.ProjectRoot)
		if err != nil {
			return err
		}
		if key != group {
			return fmt.Errorf("session project does not match directory: %s", path)
		}
		updated, err := time.Parse(time.RFC3339Nano, c.Updated)
		if err != nil {
			return err
		}
		recent = append(recent, entry{c, updated})
		if len(recent) > offset+limit+128 {
			sort.Slice(recent, newer)
			clear(recent[offset+limit:])
			recent = recent[:offset+limit]
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(recent, newer)
	for i := offset; i < min(len(recent), offset+limit); i++ {
		out = append(out, recent[i].conversation)
	}
	return out, nil
}
