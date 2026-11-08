// Purpose: Bound new session allocations while preserving existing conversations and their backups.
// Pattern: Project-owned durable allocation ceiling.
// Usage: Sessions.Create reserves a slot before publishing a new database.
// Related: internal/storage/sqlite/catalog.go, internal/storage/capacity/reserve.go.
package sqlite

import (
	"context"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/storage/capacity"
)

const maxProjectSessions = 4096

func (s *Sessions) reserveSession(ctx context.Context, directory, project string) (*capacity.Reservation, error) {
	return capacity.Reserve(ctx, filepath.Join(directory, "capacity"), maxProjectSessions, func() (count int64, err error) {
		err = s.walkDatabases(ctx, project, func(string, string, string) error {
			count++
			if count >= maxProjectSessions {
				return capacity.ErrLimit
			}
			return nil
		})
		return count, err
	})
}
