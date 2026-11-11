// Purpose: Own private, bounded file-effect receipt databases without modifying conversation formats.
// Pattern: Atomic SQLite publication and one owned connection.
// Usage: The server composition root opens one ledger per canonical workspace and closes it last.
// Related: internal/storage/filemutations/ledger.go, internal/config/statefs/files.go.
package filemutations

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"net/url"
	"os"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	_ "modernc.org/sqlite"
)

const schema = `PRAGMA application_id=1296388682; PRAGMA user_version=1;
CREATE TABLE receipts(id BLOB PRIMARY KEY CHECK(length(id)=32),digest BLOB NOT NULL CHECK(length(digest)=32),charge INTEGER NOT NULL CHECK(charge BETWEEN 0 AND 16384),body BLOB NOT NULL CHECK(length(body)<=16384));
CREATE TABLE budget(id INTEGER PRIMARY KEY CHECK(id=1),entries INTEGER NOT NULL,bytes INTEGER NOT NULL);
INSERT INTO budget VALUES(1,0,0);`

type Ledger struct{ db *sql.DB }

func connect(ctx context.Context, path string) (*sql.DB, error) {
	u := url.URL{Scheme: "file", Path: path}
	query := url.Values{}
	for _, pragma := range []string{"busy_timeout(5000)", "synchronous(FULL)", "cache_size(-1024)", "wal_autocheckpoint(128)", "max_page_count(16384)"} {
		query.Add("_pragma", pragma)
	}
	u.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err = db.PingContext(ctx); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return db, nil
}
func Open(ctx context.Context, directory string) (result *Ledger, resultErr error) {
	defer func() {
		if resultErr != nil && result != nil {
			resultErr = errors.Join(resultErr, result.Close())
			result = nil
		}
	}()
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	root, err := statefs.Open(absolute)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, lease.Close()) }()
	const name = "mutations.db"
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		temporary, err := os.CreateTemp(absolute, ".mutations-*.db")
		if err != nil {
			return nil, err
		}
		path := temporary.Name()
		defer func() {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				resultErr = errors.Join(resultErr, err)
			}
		}()
		if err := temporary.Close(); err != nil {
			return nil, err
		}
		db, err := connect(ctx, path)
		if err != nil {
			return nil, err
		}
		_, createErr := db.ExecContext(ctx, schema)
		if err := errors.Join(createErr, db.Close()); err != nil {
			return nil, err
		}
		if err := os.Link(path, filepath.Join(absolute, name)); err != nil {
			return nil, err
		}
		directory, err := os.Open(absolute)
		if err != nil {
			return nil, err
		}
		if err := errors.Join(directory.Sync(), directory.Close()); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if !info.Mode().IsRegular() {
		return nil, fault.New(diag.IntegrityFailed, diag.FileMutationStorage)
	}
	// Check the published format before any writable SQLite connection can set
	// pragmas or create journals. Unsupported user data must remain untouched.
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	var header [100]byte
	_, readErr := file.ReadAt(header[:], 0)
	if err := errors.Join(readErr, file.Close()); err != nil {
		return nil, err
	}
	if string(header[:16]) != "SQLite format 3\x00" || binary.BigEndian.Uint32(header[60:64]) != 1 || binary.BigEndian.Uint32(header[68:72]) != 1296388682 {
		return nil, fault.New(diag.IntegrityFailed, diag.FileMutationStorage)
	}
	db, err := connect(ctx, filepath.Join(absolute, name))
	if err != nil {
		return nil, err
	}
	valid := false
	defer func() {
		if !valid {
			resultErr = errors.Join(resultErr, db.Close())
		}
	}()
	var application, version int
	if err := db.QueryRowContext(ctx, "PRAGMA application_id").Scan(&application); err != nil {
		return nil, err
	}
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return nil, err
	}
	if application != 1296388682 || version != 1 {
		return nil, fault.New(diag.IntegrityFailed, diag.FileMutationStorage)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}
	valid = true
	return &Ledger{db: db}, nil
}
func (l *Ledger) Close() error { return l.db.Close() }
