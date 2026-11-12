// Purpose: Reuse one read-only SQLite connection across independent session database files.
// Pattern: Scoped I/O adapter.
// Usage: Each catalog worker owns and closes one reader; no session records are copied.
// Related: internal/storage/sqlite/conversation_scan.go, internal/storage/sqlite/session.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type conversationResult struct {
	conversation sessionstate.Conversation
	updated      time.Time
}

type conversationReader struct {
	db         *sql.DB
	connection *sql.Conn
}

func openConversationReader(ctx context.Context) (*conversationReader, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &conversationReader{}, nil
}

func (r *conversationReader) open(ctx context.Context) error {
	if r.db != nil {
		return nil
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)
	connection, err := db.Conn(ctx)
	if err != nil {
		return errors.Join(err, db.Close())
	}
	r.db, r.connection = db, connection
	if _, err := connection.ExecContext(ctx, "PRAGMA busy_timeout=5000"); err != nil {
		return errors.Join(err, r.Close())
	}
	return nil
}

func (r *conversationReader) Close() error {
	if r.db == nil {
		return nil
	}
	err := errors.Join(r.connection.Close(), r.db.Close())
	r.db, r.connection = nil, nil
	return err
}

func (r *conversationReader) Read(ctx context.Context, file conversationFile) (conversationResult, error) {
	var value conversationResult
	if err := ctx.Err(); err != nil {
		return value, err
	}
	info, err := os.Lstat(file.path)
	if errors.Is(err, os.ErrNotExist) {
		return value, nil
	}
	if err != nil {
		return value, err
	}
	if !info.Mode().IsRegular() {
		return value, faults.Errorf(diagcatalog.SqliteSessionDatabaseMustBeARegularFile, file.path)
	}
	stamp, err := conversationStamp(file.path, info)
	if err != nil {
		return value, err
	}
	if cached, ok := catalogMetadata.get(file.path, stamp, time.Now()); ok {
		return cached, nil
	}
	if err := r.open(ctx); err != nil {
		return value, err
	}
	endpoint := url.URL{Scheme: "file", Path: file.path}
	query := endpoint.Query()
	query.Set("mode", "ro")
	endpoint.RawQuery = query.Encode()
	if _, err := r.connection.ExecContext(ctx, "ATTACH DATABASE ? AS candidate", endpoint.String()); err != nil {
		return value, faults.Errorf(diagcatalog.SqliteRead, file.path, err)
	}
	value, err = r.readAttached(ctx, file.id, file.group)
	if err != nil {
		return value, faults.Errorf(diagcatalog.SqliteRead, file.path, err)
	}
	// Detach releases the file even after cancellation. SQLite's busy timeout
	// bounds cleanup; the worker closes its connection on any detach failure.
	if _, err := r.connection.ExecContext(context.WithoutCancel(ctx), "DETACH DATABASE candidate"); err != nil {
		return value, err
	}
	if after, err := os.Lstat(file.path); err == nil {
		if current, err := conversationStamp(file.path, after); err == nil && current == stamp {
			catalogMetadata.put(file.path, stamp, value, time.Now())
		}
	}
	return value, nil
}

func (r *conversationReader) readAttached(ctx context.Context, id, group string) (conversationResult, error) {
	var version int
	if err := r.connection.QueryRowContext(ctx, "PRAGMA candidate.user_version").Scan(&version); err != nil {
		return conversationResult{}, err
	}
	if version != schemaVersion {
		if !sessionstate.ValidID(id) {
			return conversationResult{}, faults.Errorf(diagcatalog.SqliteInvalidStoredConversationIdentity)
		}
		at := time.Time{}
		return conversationResult{conversation: sessionstate.Conversation{Session: sessionstate.Session{ID: id, DisplayRoot: group}, Summary: "Unavailable saved conversation", Updated: at.Format(time.RFC3339Nano), Unavailable: fmt.Sprintf("unsupported saved schema %d; required %d; original preserved", version, schemaVersion)}, updated: at}, nil
	}

	c, err := parseConversation(r.connection.QueryRowContext(ctx, `SELECT id,project_root,display_root,updated_at,COALESCE(summary,'') FROM candidate.sessions WHERE singleton=1`), id)
	if err != nil {
		return conversationResult{}, err
	}
	key, err := sessionstate.ProjectKey(c.ProjectRoot)
	if err != nil {
		return conversationResult{}, err
	}
	if key != group {
		return conversationResult{}, faults.Errorf(diagcatalog.SqliteSessionProjectDoesNotMatchDirectory)
	}
	updated, err := time.Parse(time.RFC3339Nano, c.Updated)
	return conversationResult{c, updated}, err
}
