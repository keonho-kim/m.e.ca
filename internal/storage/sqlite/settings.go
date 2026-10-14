// Purpose: Persist and parse session Permission and Scope overrides.
// Pattern: Repository.
// Usage: Session restoration and settings changes call Settings and SaveSettings.
// Related: internal/policy/policy.go, internal/tui/session/settings.go, internal/storage/sqlite/store_test.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (store *Store) Settings(ctx context.Context) (policy.Settings, error) {
	var permission, scope string
	err := store.db.QueryRowContext(ctx, "SELECT permission,scope FROM settings WHERE singleton=1").Scan(&permission, &scope)
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

func (store *Store) SaveSettings(ctx context.Context, settings policy.Settings) error {
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
	_, err := store.db.ExecContext(ctx, `INSERT INTO settings(singleton,permission,scope) VALUES(1,?,?)
ON CONFLICT(singleton) DO UPDATE SET permission=excluded.permission,scope=excluded.scope`, settings.Permission, settings.Scope)
	return err
}
