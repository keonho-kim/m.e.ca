// Purpose: Serialize foreground credential writes and reject impossible saves before OS publication.
// Pattern: Cancellable filesystem lease around repository writes.
// Usage: Put and Delete share this lease; maintenance retains its optimistic replacement boundary.
// Related: internal/storage/credentialstore/store.go, internal/config/statefs/lock.go.
package credentialstore

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/storage/capacity"
)

func (s *Store) write(ctx context.Context, apply func() error) (resultErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := statefs.Open(filepath.Join(s.directory, "credential-write"))
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, lease.Close()) }()
	if err := ctx.Err(); err != nil {
		return err
	}
	return apply()
}

func (s *Store) checkWrite(ctx context.Context, ref, scope string) error {
	var existing string
	err := s.db.QueryRowContext(ctx, "SELECT scope FROM credentials WHERE ref=?", ref).Scan(&existing)
	if err == nil {
		if existing != scope {
			return ErrIntegrity
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM credentials").Scan(&count); err != nil {
		return err
	}
	if count >= maxCredentials {
		return capacity.ErrLimit
	}
	return nil
}
