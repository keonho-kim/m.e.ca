// Purpose: Read and rewrite project file selections without restoring conversation runtimes.
// Pattern: Workspace-scoped repository over existing session databases.
// Usage: Server file operations update affected conversations with one bounded handle at a time.
// Related: internal/storage/sqlite/files.go, internal/storage/sqlite/file_selection_changes.go.
package sqlite

import (
	"context"
	"errors"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"io"
	"os"
	"strings"
)

type ProjectReferences struct {
	Sessions *Sessions
	Project  string
}

func (r ProjectReferences) List(ctx context.Context, session string) (paths []string, resultErr error) {
	store, err := r.Sessions.OpenRead(ctx, r.Project, session)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, store.Close()) }()
	return store.FileSelections(ctx, session)
}
func (r ProjectReferences) Select(ctx context.Context, session, path string, selected bool) (resultErr error) {
	store, err := r.Sessions.Open(ctx, r.Project, session)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, store.Close()) }()
	return store.SelectFile(ctx, session, path, selected)
}
func (r ProjectReferences) Rewrite(ctx context.Context, transform func(context.Context, string) (string, error), changed func(string, []string)) error {
	directory, _, err := r.Sessions.projectDirectory(r.Project, false)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	file, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer file.Close()
	// Reference rewrites need IDs only. Do not populate paginated conversation
	// snapshots or read each conversation's metadata before opening its selections.
	for {
		entries, readErr := file.ReadDir(128)
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if !strings.HasSuffix(entry.Name(), ".db") {
				continue
			}
			id := strings.TrimSuffix(entry.Name(), ".db")
			if !sessionstate.ValidID(id) {
				continue
			}
			if err := r.rewriteSession(ctx, id, transform, changed); err != nil && !errors.Is(err, sessionstate.ErrNotFound) {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (r ProjectReferences) rewriteSession(ctx context.Context, session string, transform func(context.Context, string) (string, error), changed func(string, []string)) (resultErr error) {
	store, err := r.Sessions.Open(ctx, r.Project, session)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, store.Close()) }()
	paths, err := store.FileSelections(ctx, session)
	if err != nil {
		return err
	}
	updates := make(map[string]string)
	for _, path := range paths {
		next, err := transform(ctx, path)
		if err != nil {
			return err
		}
		if next != path {
			updates[path] = next
		}
	}
	if len(updates) == 0 {
		return nil
	}
	paths, err = store.ChangeFileSelections(ctx, session, updates)
	if err != nil {
		return err
	}
	changed(session, paths)
	return nil
}
