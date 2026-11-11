// Purpose: Own one database connection, codec and handle-scoped cache/deletion leases.
// Pattern: Repository lifecycle boundary.
// Usage: Sessions opens a Store for the selected session; the caller closes it after workers stop.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/session.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	_ "modernc.org/sqlite"
)

type Store struct {
	closeOnce     sync.Once
	closeErr      error
	deletionLease *SessionLease
	teamContent   teamContentCache
	latestEvent   atomic.Int64
	room          *Store
	teamLease     *teamContentCache
	codec         recordcontent.Codec
	objective     objectiveCache
	usage         *usageProjection
	localUsage    usageProjection
	db            *sql.DB
	path          string
	session       sessionstate.Session
}

func (s *Store) Session() sessionstate.Session { return s.session }
func (s *Store) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = errors.Join(s.db.Close(), s.codec.Close())
		if s.deletionLease != nil {
			s.closeErr = errors.Join(s.closeErr, s.deletionLease.Close())
		}
		if s.teamLease != nil {
			s.teamLease.release()
			s.teamLease = nil
		}
		s.teamContent.clear()
	})
	return s.closeErr
}

func connect(ctx context.Context, path string, readOnly bool) (*sql.DB, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, faults.Errorf(diagcatalog.SqliteSessionDatabaseMustBeARegularFile, path)
	}
	endpoint := url.URL{Scheme: "file", Path: path}
	query := endpoint.Query()
	query.Set("mode", "rw")
	if readOnly {
		query.Set("mode", "ro")
	} else if err := os.Chmod(path, 0600); err != nil {
		return nil, err
	}
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "foreign_keys(1)")
	if !readOnly {
		query.Set("_txlock", "immediate")
		// SQLite enforces the main-file ceiling atomically across all handles.
		pages, err := sessionPageLimit(path)
		if err != nil {
			return nil, err
		}
		query.Add("_pragma", "max_page_count("+strconv.FormatInt(pages, 10)+")")
		query.Add("_pragma", "journal_size_limit(16777216)")
		query.Add("_pragma", "wal_autocheckpoint(256)")
	}
	endpoint.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", endpoint.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return db, nil
}

func checkSchema(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version != schemaVersion {
		return faults.Errorf(diagcatalog.SqliteUnsupportedSessionDatabaseSchemaRequiredExistingContents, version, schemaVersion)
	}
	return nil
}
