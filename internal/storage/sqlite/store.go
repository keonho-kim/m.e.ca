// Purpose: Open private SQLite storage and apply atomic schema migrations.
// Pattern: Repository.
// Usage: Opened by cmd/v; closed after all task and terminal work finishes.
// Related: internal/storage/sqlite/schema.go, internal/storage/sqlite/session.go, internal/storage/sqlite/store_test.go.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db        *sql.DB
	directory string
}

func Open(path string) (*Store, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if info, err := os.Lstat(absolute); err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("database must be a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	file, err := os.OpenFile(absolute, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := errors.Join(file.Chmod(0o600), file.Close()); err != nil {
		return nil, err
	}
	endpoint := url.URL{Scheme: "file", Path: absolute}
	query := endpoint.Query()
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "foreign_keys(1)")
	query.Set("_txlock", "immediate")
	endpoint.RawQuery = query.Encode()
	database, err := sql.Open("sqlite", endpoint.String())
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	store := &Store{db: database, directory: filepath.Dir(absolute)}
	if err := store.migrate(context.Background()); err != nil {
		return nil, errors.Join(err, database.Close())
	}
	return store, nil
}

func (store *Store) Close() error { return store.db.Close() }

func (store *Store) migrate(ctx context.Context) (err error) {
	lock, err := store.migrationLock()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	if _, err := store.db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		return err
	}

	transaction, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var version int
	if err := transaction.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return errors.Join(err, transaction.Rollback())
	}
	if version > 4 {
		return errors.Join(fmt.Errorf("database schema is newer than this V version"), transaction.Rollback())
	}
	if version == 4 {
		return transaction.Commit()
	}
	if _, err := transaction.ExecContext(ctx, migration(version)); err != nil {
		return errors.Join(err, transaction.Rollback())
	}
	return transaction.Commit()
}
