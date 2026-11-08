// Purpose: Bound new OS encryption-key versions while preserving previous backup keys.
// Pattern: Durable conservative key-allocation budget.
// Usage: seal reserves before attempting OS publication, including uncertain failures.
// Related: internal/storage/credentialstore/crypto.go, internal/storage/capacity/reserve.go.
package credentialstore

import (
	"context"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/storage/capacity"
)

const maxCredentials = 256
const maxKeyVersions = 65536

func (s *Store) reserveKey(ctx context.Context) (*capacity.Reservation, error) {
	return capacity.Reserve(ctx, filepath.Join(s.directory, "credential-key-capacity"), maxKeyVersions, func() (count int64, err error) {
		err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM credentials").Scan(&count)
		return count, err
	})
}
