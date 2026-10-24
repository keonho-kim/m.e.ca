// Purpose: Reuse one read-only SQLite reader while discovering independent session database files.
// Pattern: Scoped I/O adapter.
// Usage: Conversations attaches one file at a time and closes its reader before returning.
// Related: internal/storage/sqlite/conversations.go, internal/storage/sqlite/directories.go, internal/storage/sqlite/session.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"net/url"
	"os"
	"time"
)

type conversationResult struct {
	conversation sessionstate.Conversation
	updated      time.Time
}

func (s *Sessions) scanConversations(ctx context.Context, project string, accept func(conversationResult)) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	// This private in-memory connection holds no registry or session records.
	// ATTACH reuses driver initialization and has exactly one file open at a time.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, db.Close()) }()
	db.SetMaxOpenConns(1)
	reader, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, reader.Close()) }()
	if _, err := reader.ExecContext(ctx, "PRAGMA busy_timeout=5000"); err != nil {
		return err
	}
	return s.walkDatabases(ctx, project, func(path, id, group string) error {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("session database must be a regular file: %s", path)
		}
		endpoint := url.URL{Scheme: "file", Path: path}
		query := endpoint.Query()
		query.Set("mode", "ro")
		endpoint.RawQuery = query.Encode()
		if _, err := reader.ExecContext(ctx, "ATTACH DATABASE ? AS candidate", endpoint.String()); err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		result, err := readAttachedConversation(ctx, reader, id, group)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if _, err := reader.ExecContext(ctx, "DETACH DATABASE candidate"); err != nil {
			return err
		}
		accept(result)
		return nil
	})
}

func readAttachedConversation(ctx context.Context, reader *sql.Conn, id, group string) (conversationResult, error) {
	var version int
	if err := reader.QueryRowContext(ctx, "PRAGMA candidate.user_version").Scan(&version); err != nil {
		return conversationResult{}, err
	}
	if version != schemaVersion {
		return conversationResult{}, fmt.Errorf("unsupported session database schema %d; required %d; existing contents are preserved", version, schemaVersion)
	}
	c, err := parseConversation(reader.QueryRowContext(ctx, `SELECT id,project_root,display_root,updated_at,COALESCE(summary,'New conversation') FROM candidate.sessions WHERE singleton=1`), id)
	if err != nil {
		return conversationResult{}, err
	}
	key, err := sessionstate.ProjectKey(c.ProjectRoot)
	if err != nil {
		return conversationResult{}, err
	}
	if key != group {
		return conversationResult{}, fmt.Errorf("session project does not match directory")
	}
	updated, err := time.Parse(time.RFC3339Nano, c.Updated)
	return conversationResult{c, updated}, err
}
