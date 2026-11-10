// Purpose: Delete a leased public conversation and its private team files with restartable intent.
// Pattern: Filesystem tombstone and exact repository-owned cleanup.
// Usage: Native deleteSession closes owned readers before Sessions.DeleteConversation.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/team_sessions.go.
package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func (s *Sessions) DeleteConversation(ctx context.Context, project, id string) (resultErr error) {
	if !sessionstate.ValidID(id) {
		return fault.New(diag.InvalidArgument, diag.ServerExplicitSession)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	directory, _, err := s.projectDirectory(project, false)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	lease, err := openConversationDeletionLock(filepath.Join(directory, id+".db.lock"))
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, lease.Close()) }()
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	marker := filepath.Join(".deleted", id)
	_, pending := root.Stat(marker)
	if errors.Is(pending, os.ErrNotExist) {
		store, err := s.OpenRead(ctx, project, id)
		if errors.Is(err, sessionstate.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := store.Close(); err != nil {
			return err
		}
	} else if pending != nil {
		return pending
	}
	guard, err := openConversationDeletionLock(filepath.Join(directory, id+".db.guard"))
	if err != nil {
		return fault.New(diag.Conflict, diag.UIDeleteBusy)
	}
	defer func() { resultErr = errors.Join(resultErr, guard.Close()) }()
	if err := root.MkdirAll(marker, 0700); err != nil {
		return err
	}
	// Persist intent before unlinking data, and data removal before removing
	// intent. Startup can safely finish either interrupted phase.
	if err := syncDeletionDirectory(root, ".deleted"); err != nil {
		return err
	}
	if err := syncDeletionDirectory(root, "."); err != nil {
		return err
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := root.Remove(id + ".db" + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := root.RemoveAll(filepath.Join(".team", id)); err != nil {
		return err
	}
	if err := syncDeletionDirectory(root, ".team"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := syncDeletionDirectory(root, "."); err != nil {
		return err
	}
	if err := root.Remove(marker); err != nil {
		return err
	}
	return syncDeletionDirectory(root, ".deleted")
}

func syncDeletionDirectory(root *os.Root, path string) error {
	directory, err := root.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

func (s *Sessions) CleanupDeleted(ctx context.Context, project string) error {
	directory, _, err := s.projectDirectory(project, false)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(directory, ".deleted"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() && sessionstate.ValidID(entry.Name()) {
			if err := s.DeleteConversation(ctx, project, entry.Name()); err != nil {
				return err
			}
		}
	}
	return nil
}
