// Purpose: Restore project-scoped sessions and persist validated policy overrides.
// Pattern: Repository.
// Usage: Called during startup and project Permission/Scope changes.
// Related: internal/storage/sqlite/store.go, internal/storage/sqlite/store_test.go, internal/policy/policy.go.

package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/keonho-kim/v-code/internal/policy"
)

type Session struct {
	ID          string
	ProjectRoot string
	DisplayRoot string
}

func (store *Store) RestoreSession(ctx context.Context, project, display string, restore bool) (Session, error) {
	if !filepath.IsAbs(project) || len(project) > 4096 || display == "" {
		return Session{}, fmt.Errorf("invalid project root")
	}
	project = filepath.Clean(project)
	session := Session{ProjectRoot: project, DisplayRoot: display}
	if restore {
		err := store.db.QueryRowContext(ctx, "SELECT id, display_root FROM sessions WHERE project_root=? ORDER BY updated_at DESC, rowid DESC LIMIT 1", project).Scan(&session.ID, &session.DisplayRoot)
		if err == nil {
			return session, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return session, err
		}
	}
	session.ID = "session_" + rand.Text()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := store.db.ExecContext(ctx, "INSERT INTO sessions(id,project_root,display_root,created_at,updated_at) VALUES(?,?,?,?,?)", session.ID, project, display, now, now)
	return session, err
}

func (store *Store) ProjectSettings(ctx context.Context, project string) (policy.Settings, error) {
	var permission, scope string
	err := store.db.QueryRowContext(ctx, "SELECT permission,scope FROM project_settings WHERE project_root=?", project).Scan(&permission, &scope)
	if errors.Is(err, sql.ErrNoRows) {
		return policy.Settings{}, nil
	}
	if err != nil {
		return policy.Settings{}, err
	}
	settings := policy.Settings{}
	if permission != "" {
		settings.Permission, err = policy.ParsePermission(permission)
		if err != nil {
			return settings, err
		}
	}
	if scope != "" {
		settings.Scope, err = policy.ParseScope(scope)
	}
	return settings, err
}

func (store *Store) SaveProjectSettings(ctx context.Context, project string, settings policy.Settings) error {
	if !filepath.IsAbs(project) {
		return fmt.Errorf("project root must be absolute")
	}
	if settings.Permission != "" {
		if _, err := policy.ParsePermission(string(settings.Permission)); err != nil {
			return err
		}
	}
	if settings.Scope != "" {
		if _, err := policy.ParseScope(string(settings.Scope)); err != nil {
			return err
		}
	}
	_, err := store.db.ExecContext(ctx, `INSERT INTO project_settings(project_root,permission,scope) VALUES(?,?,?)
ON CONFLICT(project_root) DO UPDATE SET permission=excluded.permission,scope=excluded.scope`, project, settings.Permission, settings.Scope)
	return err
}
