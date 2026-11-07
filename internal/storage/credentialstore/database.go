// Purpose: Open or atomically publish a private, versioned credential database without replacing unsupported data.
// Pattern: SQLite resource lifecycle and filesystem trust boundary.
// Usage: config/credentials supplies its private state directory and OS key adapter.
// Related: internal/config/statefs/root.go, internal/storage/credentialstore/store.go.
package credentialstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/config/statefs"
	_ "modernc.org/sqlite"
)

const Filename = "credentials.db"
const schema = `CREATE TABLE credentials(
ref TEXT PRIMARY KEY,
scope TEXT NOT NULL CHECK(length(scope) BETWEEN 1 AND 128),
key_ref TEXT NOT NULL CHECK(length(key_ref)=37),
sealed BLOB NOT NULL CHECK(length(sealed) BETWEEN 29 AND 2076)
);
PRAGMA user_version=1;
PRAGMA application_id=1296384833;`

func Open(ctx context.Context, directory string, keys Keys, create bool) (store *Store, resultErr error) {
	if keys == nil {
		return nil, fmt.Errorf("OS key storage is required")
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	root, err := statefs.Open(directory)
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, root.Close())
		if resultErr != nil && store != nil {
			resultErr = errors.Join(resultErr, store.Close())
			store = nil
		}
	}()
	if _, err := root.Lstat(Filename); errors.Is(err, os.ErrNotExist) && create {
		if err := publish(ctx, root, directory); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	db, err := connect(ctx, filepath.Join(directory, Filename))
	if err != nil {
		return nil, err
	}
	var version, application int
	err = db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	if err == nil {
		err = db.QueryRowContext(ctx, "PRAGMA application_id").Scan(&application)
	}
	if err == nil && (version != 1 || application != 1296384833) {
		err = fmt.Errorf("unsupported credential database; existing contents were preserved")
	}
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return &Store{db: db, keys: keys, directory: directory}, nil
}

func connect(ctx context.Context, path string) (*sql.DB, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("credential database must be a private regular file")
	}
	u := url.URL{Scheme: "file", Path: path}
	q := u.Query()
	q.Set("mode", "rw")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "synchronous(FULL)")
	q.Add("_pragma", "secure_delete(1)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return db, nil
}

func publish(ctx context.Context, root *os.Root, directory string) (resultErr error) {
	file, err := os.CreateTemp(directory, ".pending-credentials-*.db")
	if err != nil {
		return err
	}
	name := filepath.Base(file.Name())
	defer func() { resultErr = errors.Join(resultErr, root.Remove(name)) }()
	if err := file.Close(); err != nil {
		return err
	}
	db, err := connect(ctx, file.Name())
	if err != nil {
		return err
	}
	_, writeErr := db.ExecContext(ctx, schema)
	if err := errors.Join(writeErr, db.Close()); err != nil {
		return err
	}
	if err := root.Link(name, Filename); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
