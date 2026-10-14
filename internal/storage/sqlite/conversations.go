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
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *Sessions) Conversations(ctx context.Context, offset, limit int) ([]sessionstate.Conversation, error) {
	return s.conversations(ctx, "", offset, limit)
}
func (s *Sessions) conversations(ctx context.Context, project string, offset, limit int) (out []sessionstate.Conversation, err error) {
	if offset < 0 || offset > 100000 || limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid conversation page")
	}
	directory, err := os.Open(s.directory)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, directory.Close()) }()
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
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		files, readErr := directory.ReadDir(128)
		for _, file := range files {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			name := file.Name()
			if !strings.HasPrefix(name, "session_") || !strings.HasSuffix(name, ".db") {
				continue
			}
			id := strings.TrimSuffix(name, ".db")
			if !sessionstate.ValidID(id) {
				return nil, fmt.Errorf("invalid session database filename: %s", name)
			}
			db, err := connect(ctx, filepath.Join(s.directory, name), true)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", name, err)
			}
			c, readErr := readConversation(ctx, db, id)
			if err := errors.Join(readErr, db.Close()); err != nil {
				return nil, fmt.Errorf("read %s: %w", name, err)
			}
			if project != "" && c.ProjectRoot != project {
				continue
			}
			updated, err := time.Parse(time.RFC3339Nano, c.Updated)
			if err != nil {
				return nil, err
			}
			recent = append(recent, entry{c, updated})
			// Keep only the requested prefix; no unbounded history or open-handle cache.
			if len(recent) > offset+limit+128 {
				sort.Slice(recent, newer)
				clear(recent[offset+limit:])
				recent = recent[:offset+limit]
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	sort.Slice(recent, newer)
	for i := offset; i < min(len(recent), offset+limit); i++ {
		out = append(out, recent[i].conversation)
	}
	return out, nil
}
