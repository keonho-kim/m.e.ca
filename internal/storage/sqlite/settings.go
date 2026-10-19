// Purpose: Persist and parse session Permission overrides.
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
	var permission string
	err := store.db.QueryRowContext(ctx, "SELECT permission FROM settings WHERE singleton=1").Scan(&permission)
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
	return settings, err
}

func (store *Store) SaveSettings(ctx context.Context, settings policy.Settings) error {
	if settings.Permission != "" {
		if _, err := policy.ParsePermission(string(settings.Permission)); err != nil {
			return err
		}
	}
	_, err := store.db.ExecContext(ctx, `INSERT INTO settings(singleton,permission) VALUES(1,?)
ON CONFLICT(singleton) DO UPDATE SET permission=excluded.permission`, settings.Permission)
	return err
}
