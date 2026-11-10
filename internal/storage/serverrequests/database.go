// Purpose: Open one private project-scoped request database while preserving unsupported prior records.
// Pattern: Atomic database publication and process-owned SQLite lifecycle.
// Usage: cmd/meca owns Ledger until server execution and readers have joined.
// Related: internal/storage/serverrequests/ledger.go, internal/config/statefs/files.go.
package serverrequests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"

	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	_ "modernc.org/sqlite"
)

const Filename = "requests.db"
const requestSchema = `
PRAGMA application_id=1296388689;
PRAGMA user_version=4;
CREATE TABLE project(
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 scope BLOB NOT NULL CHECK(length(scope)=32)
);
CREATE TABLE requests(
 id BLOB PRIMARY KEY CHECK(length(id)=32),
 digest BLOB NOT NULL CHECK(length(digest)=32),
 session TEXT NOT NULL DEFAULT '',
 status TEXT NOT NULL DEFAULT '',
 error TEXT NOT NULL DEFAULT '',
 error_code TEXT NOT NULL DEFAULT '' CHECK(length(error_code)<=128)
);
CREATE INDEX requests_session_id ON requests(session,id);`

type Ledger struct {
	db              *sql.DB
	root            *os.Root
	directory, path string
}

func Open(ctx context.Context, directory, scope string) (ledger *Ledger, resultErr error) {
	defer func() {
		if resultErr != nil && ledger != nil {
			resultErr = errors.Join(resultErr, ledger.Close())
			ledger = nil
		}
	}()
	if len(scope) != 64 {
		return nil, faults.Errorf(diagcatalog.ServerrequestsRequestStorageRequiresAProjectScope)
	}
	if _, err := hex.DecodeString(scope); err != nil || strings.ToLower(scope) != scope {
		return nil, faults.Errorf(diagcatalog.ServerrequestsInvalidProjectScope)
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	parent, err := statefs.Open(directory)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, parent.Close()) }()
	ancestor, err := os.Open(filepath.Dir(directory))
	if err != nil {
		return nil, err
	}
	if err := errors.Join(ancestor.Sync(), ancestor.Close()); err != nil {
		return nil, err
	}
	lease, err := statefs.Lock(ctx, parent)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, lease.Close()) }()
	if err := rejectLegacy(parent); err != nil {
		return nil, err
	}

	folder := filepath.Join(directory, scope)
	root, err := statefs.Open(folder)
	if err != nil {
		return nil, err
	}
	defer func() {
		if ledger == nil {
			resultErr = errors.Join(resultErr, root.Close())
		}
	}()
	dir, err := parent.Open(".")
	if err != nil {
		return nil, err
	}
	if err := errors.Join(dir.Sync(), dir.Close()); err != nil {
		return nil, err
	}
	path := filepath.Join(folder, Filename)
	if _, err := root.Lstat(Filename); errors.Is(err, os.ErrNotExist) {
		if err := publish(ctx, root, folder, scope); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	db, err := connect(ctx, path)
	if err != nil {
		return nil, err
	}
	var version, application int
	err = db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	if err == nil {
		err = db.QueryRowContext(ctx, "PRAGMA application_id").Scan(&application)
	}
	if err == nil && (version != 4 || application != 1296388689) {
		err = faults.Errorf(diagcatalog.ServerrequestsUnsupportedRequestDatabaseExistingContentsPreserved)
	}
	if err == nil {
		var storedScope []byte
		err = db.QueryRowContext(ctx, "SELECT scope FROM project WHERE singleton=1").Scan(&storedScope)
		expected, _ := hex.DecodeString(scope)
		if err == nil && !bytes.Equal(storedScope, expected) {
			err = faults.Errorf(diagcatalog.ServerrequestsRequestDatabaseBelongsToAnotherProjectOriginal)
		}
	}
	if err == nil {
		_, err = db.ExecContext(ctx, "PRAGMA journal_mode=WAL")
	}
	if err == nil {
		_, err = db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS requests_session_id ON requests(session,id)")
	}
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return &Ledger{db: db, root: root, directory: directory, path: path}, nil
}
func (l *Ledger) Close() error { return errors.Join(l.db.Close(), l.root.Close()) }

func connect(ctx context.Context, path string) (*sql.DB, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, faults.Errorf(diagcatalog.ServerrequestsRequestDatabaseMustBeAPrivateRegular)
	}
	u := url.URL{Scheme: "file", Path: path}
	q := u.Query()
	q.Set("mode", "rw")
	q.Set("_txlock", "immediate")
	for _, pragma := range []string{"busy_timeout(5000)", "synchronous(FULL)", "wal_autocheckpoint(256)", "journal_size_limit(16777216)"} {
		q.Add("_pragma", pragma)
	}
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
func publish(ctx context.Context, root *os.Root, directory, scope string) (resultErr error) {
	file, err := os.CreateTemp(directory, ".pending-requests-*.db")
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
	_, writeErr := db.ExecContext(ctx, requestSchema)
	if writeErr == nil {
		digest, _ := hex.DecodeString(scope)
		_, writeErr = db.ExecContext(ctx, "INSERT INTO project(singleton,scope) VALUES(1,?)", digest)
	}
	if err := errors.Join(writeErr, db.Close()); err != nil {
		return err
	}
	if err := root.Link(name, Filename); err != nil {
		return err
	}
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func rejectLegacy(root *os.Root) (resultErr error) {
	file, err := root.Open(".")
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	for {
		entries, err := file.ReadDir(128)
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".json") {
				return faults.Errorf(diagcatalog.ServerrequestsUnsupportedJSONRequestLedgerExistingRecordsWere)
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
