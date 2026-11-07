// Purpose: Re-encrypt due credentials without overwriting concurrent saves or losing prior OS keys.
// Pattern: Serialized maintenance lease and optimistic per-record replacement.
// Usage: The process-owned credential worker supplies the current time and internal cryptoperiod.
// Related: internal/storage/credentialstore/rotation_schedule.go, internal/storage/credentialstore/crypto.go.
package credentialstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/keonho-kim/meca/internal/config/statefs"
)

type credentialVersion struct {
	ref, scope, keyRef string
	sealed             []byte
}

// RotateDue leaves API key values and saved connection references unchanged.
// New or externally replaced versions begin their period on first observation.
func (s *Store) RotateDue(ctx context.Context, now time.Time, period time.Duration) (resultErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if now.UnixNano() <= 0 || period <= 0 {
		return fmt.Errorf("invalid credential rotation period or time")
	}
	directory := filepath.Join(s.directory, rotationDirectory)
	root, err := statefs.Open(directory)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	lease, err := statefs.Lease(root, true)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return nil // Another process owns this maintenance pass.
	}
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, lease.Close()) }()
	var first error
	failures := 0
	cursor, err := rotationCursor(root)
	if err != nil {
		return err
	}
	after := cursor
	for {
		page, err := s.rotationPage(ctx, after)
		if err != nil {
			return checkpointRotation(ctx, directory, after, errors.Join(first, err))
		}
		if len(page) == 0 {
			if cursor != "" {
				if err := writeRotationCursor(directory, ""); err != nil {
					return errors.Join(first, err)
				}
			}
			break
		}
		for _, row := range page {
			if err := ctx.Err(); err != nil {
				return checkpointRotation(ctx, directory, after, errors.Join(first, err))
			}
			after = row.ref
			if err := s.rotateVersion(ctx, root, directory, row, now, period); err != nil {
				failures++
				if first == nil {
					first = err
				}
			}
		}
	}
	if first != nil {
		return fmt.Errorf("credential rotation deferred for %d records: %w", failures, first)
	}
	return nil
}

func (s *Store) rotationPage(ctx context.Context, after string) (page []credentialVersion, resultErr error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ref,scope,key_ref,
CASE WHEN length(sealed) BETWEEN 29 AND 2076 THEN sealed END
FROM credentials WHERE ref>? ORDER BY ref LIMIT 32`, after)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, rows.Close()) }()
	for rows.Next() {
		var row credentialVersion
		if err := rows.Scan(&row.ref, &row.scope, &row.keyRef, &row.sealed); err != nil {
			return nil, err
		}
		page = append(page, row)
	}
	return page, rows.Err()
}

func (s *Store) rotateVersion(ctx context.Context, root *os.Root, directory string, row credentialVersion, now time.Time, period time.Duration) error {
	if err := checkIdentity(row.ref, row.scope); err != nil {
		return err
	}
	if len(row.keyRef) != 37 || !strings.HasPrefix(row.keyRef, "encryption_") || len(row.sealed) < 29 {
		return ErrIntegrity
	}
	due, err := rotationDue(root, directory, row, now, period)
	if err != nil || !due {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	secret, err := s.open(ctx, row.ref, row.scope, row.keyRef, row.sealed)
	if err != nil {
		return err
	}
	next, sealed, err := s.seal(ctx, row.ref, row.scope, secret)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE credentials SET key_ref=?,sealed=?
WHERE ref=? AND scope=? AND key_ref=? AND sealed=?`, next, sealed, row.ref, row.scope, row.keyRef, row.sealed)
	if err != nil {
		return err // Uncertain commits retain both OS keys.
	}
	changed, err := result.RowsAffected()
	if err != nil || changed == 0 {
		return err // A concurrent replacement/deletion wins without replay.
	}
	return recordRotation(directory, row.ref, next, now)
}
