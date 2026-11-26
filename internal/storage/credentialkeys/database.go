// Purpose: Open or atomically initialize the versioned database of local encryption keys.
// Pattern: Private SQLite resource lifecycle with immutable database identity.
// Usage: Store.Set initializes storage and Store.Get opens existing storage with crash recovery.
// Related: internal/storage/credentialkeys/files.go, internal/storage/credentialkeys/store.go.
package credentialkeys

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	_ "modernc.org/sqlite"
)

const applicationID = 1296384834
const schema = `PRAGMA page_size=4096;
BEGIN IMMEDIATE;
CREATE TABLE keys(ref TEXT PRIMARY KEY CHECK(length(ref)=37),
material BLOB NOT NULL CHECK(typeof(material)='blob' AND length(material)=32)) WITHOUT ROWID;
PRAGMA user_version=1;
PRAGMA application_id=1296384834;
COMMIT;`

func (s Store) open(ctx context.Context, create bool) (db *sql.DB, resultErr error) {
	root, err := s.root(create)
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, root.Close())
		if resultErr != nil && db != nil {
			resultErr = errors.Join(resultErr, db.Close())
			db = nil
		}
	}()
	if _, err := root.Lstat(Filename); errors.Is(err, os.ErrNotExist) && create {
		if err := publish(ctx, root); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	db, err = connect(ctx, root, Filename)
	if err != nil {
		return nil, err
	}
	var version, application, pageSize int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return db, err
	}
	if err := db.QueryRowContext(ctx, "PRAGMA application_id").Scan(&application); err != nil {
		return db, err
	}
	if err := db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return db, err
	}
	if version != 1 || application != applicationID || pageSize != 4096 {
		return db, fault.Errorf(diag.LocalKeysUnsupportedStore)
	}
	return db, nil
}

func connect(ctx context.Context, root *os.Root, name string) (*sql.DB, error) {
	if err := privateFile(root, name); err != nil {
		return nil, err
	}
	path, err := filepath.Abs(filepath.Join(root.Name(), name))
	if err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: path}
	query := u.Query()
	// Reads also need rollback-journal recovery after an interrupted writer.
	// mode=rw opens existing files without recreating a missing database.
	query.Set("mode", "rw")
	query.Add("_pragma", "busy_timeout(5000)")
	// EXTRA also syncs journal removal before ciphertext can reference a key.
	query.Add("_pragma", "synchronous(EXTRA)")
	// A 16 MiB ceiling bounds this fixed-size key store independently of DB backups.
	query.Add("_pragma", "max_page_count(4096)")
	u.RawQuery = query.Encode()
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

func publish(ctx context.Context, root *os.Root) (resultErr error) {
	name := ".pending-keys-" + rand.Text()
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Remove(name)) }()
	if err := file.Close(); err != nil {
		return err
	}
	db, err := connect(ctx, root, name)
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
	if err := statefs.SyncDirectory(root); err != nil {
		return err
	}
	parent, err := os.OpenRoot(filepath.Dir(root.Name()))
	if err != nil {
		return err
	}
	return errors.Join(statefs.SyncDirectory(parent), parent.Close())
}
