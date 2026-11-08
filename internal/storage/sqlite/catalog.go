// Purpose: Create and select session database files without a separate session registry.
// Pattern: Filesystem-backed session lifecycle.
// Usage: Startup opens ~/.meca/sessions, then calls Create or Open.
// Related: internal/storage/sqlite/store.go, internal/storage/sqlite/conversations.go, internal/config/statefs/root.go.
package sqlite

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type Sessions struct {
	directory string
	pages     conversationPages
}

func OpenSessions(directory string) (*Sessions, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	root, err := statefs.Open(absolute)
	if err != nil {
		return nil, err
	}
	if err := root.Close(); err != nil {
		return nil, err
	}
	return &Sessions{directory: absolute}, nil
}
func (s *Sessions) Open(ctx context.Context, project, id string) (store *Store, resultErr error) {
	return s.open(ctx, project, id, false)
}

func (s *Sessions) open(ctx context.Context, project, id string, readOnly bool) (store *Store, resultErr error) {
	defer func() {
		if errors.Is(resultErr, os.ErrNotExist) {
			resultErr = errors.Join(sessionstate.ErrNotFound, resultErr)
		}
	}()
	if !sessionstate.ValidID(id) {
		return nil, faults.Errorf(diagcatalog.SqliteInvalidSessionID)
	}
	directory, canonical, err := s.projectDirectory(project, false)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(directory, id+".db")
	db, err := connect(ctx, path, readOnly)
	if err != nil {
		return nil, err
	}
	c, err := readConversation(ctx, db, id)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	if c.ProjectRoot != canonical {
		return nil, errors.Join(faults.Errorf(diagcatalog.SqliteSessionProjectDoesNotMatchItsStorage), db.Close())
	}
	if !readOnly {
		if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
			return nil, errors.Join(err, db.Close())
		}
	}
	return &Store{db: db, path: path, session: c.Session}, nil
}
func (s *Sessions) Create(ctx context.Context, project, display string) (store *Store, err error) {
	if err := sessionstate.CheckProject(project, display); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	directory, project, err := s.projectDirectory(project, true)
	if err != nil {
		return nil, err
	}
	reservation, err := s.reserveSession(ctx, directory, project)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, reservation.Close()) }()
	id := "session_" + rand.Text()
	// Publish a complete, closed database atomically. Directory scans never see a
	// half-initialized session; no central catalogue or bootstrap lock is needed.
	temporary, err := os.CreateTemp(directory, ".pending-*.db")
	if err != nil {
		return nil, err
	}
	path := temporary.Name()
	defer func() {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
			if store != nil {
				err = errors.Join(err, store.Close())
				store = nil
			}
		}
	}()
	if err := temporary.Close(); err != nil {
		return nil, err
	}
	db, err := connect(ctx, path, false)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	if _, err := tx.ExecContext(ctx, schema); err != nil {
		return nil, errors.Join(err, tx.Rollback(), db.Close())
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO sessions(id,project_root,display_root,created_at,updated_at) VALUES(?,?,?,?,?)", id, project, display, now, now); err != nil {
		return nil, errors.Join(err, tx.Rollback(), db.Close())
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", schemaVersion)); err != nil {
		return nil, errors.Join(err, tx.Rollback(), db.Close())
	}
	if err := errors.Join(tx.Commit(), db.Close()); err != nil {
		return nil, err
	}
	reservation.Publish()
	if err := os.Link(path, filepath.Join(directory, id+".db")); err != nil {
		return nil, err
	}
	directoryFile, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	if err := errors.Join(directoryFile.Sync(), directoryFile.Close()); err != nil {
		return nil, err
	}
	return s.Open(ctx, project, id)
}
