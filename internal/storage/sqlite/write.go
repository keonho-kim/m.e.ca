// Purpose: Admit SQLite writes only when retained readers are not driving an oversized WAL.
// Pattern: Connection-owned transaction boundary with pre-effect backpressure.
// Usage: Journal and metadata writers share this connection/transaction lifetime.
// Related: internal/storage/sqlite/commit.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"github.com/keonho-kim/meca/internal/diagnostics"
	"os"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const walPressureBytes = 16 << 20

func (s *Store) beginWrite(ctx context.Context) (*sql.Tx, func(), error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, nil, err
	}
	var checked os.FileInfo
	var checkedVersion int64
	for range 3 {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return nil, nil, errors.Join(err, conn.Close())
		}
		info, statErr := os.Stat(s.path + "-wal")
		if errors.Is(statErr, os.ErrNotExist) || statErr == nil && info.Size() < walPressureBytes {
			return tx, func() { _ = tx.Rollback(); _ = conn.Close() }, nil
		}
		if statErr == nil && checked != nil && os.SameFile(info, checked) && info.Size() == checked.Size() && info.ModTime().Equal(checked.ModTime()) {
			var version int64
			if err := tx.QueryRowContext(ctx, "PRAGMA data_version").Scan(&version); err != nil {
				return nil, nil, errors.Join(err, tx.Rollback(), conn.Close())
			}
			if version == checkedVersion {
				return tx, func() { _ = tx.Rollback(); _ = conn.Close() }, nil
			}
		}
		if err := tx.Rollback(); err != nil {
			return nil, nil, errors.Join(err, conn.Close())
		}
		if statErr != nil {
			return nil, nil, errors.Join(statErr, conn.Close())
		}
		var busy, pages, backfilled int
		if err := conn.QueryRowContext(ctx, "PRAGMA data_version").Scan(&checkedVersion); err != nil {
			return nil, nil, errors.Join(err, conn.Close())
		}
		if err := conn.QueryRowContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)").Scan(&busy, &pages, &backfilled); err != nil {
			return nil, nil, errors.Join(err, conn.Close())
		}
		if busy != 0 || backfilled < pages {
			diagnostics.Log(ctx, diag.StorageCapacityLow, "component", "storage", "wal_bytes", info.Size(), "checkpoint_pages", pages, "checkpoint_backfilled", backfilled)
			break
		}
		// BEGIN IMMEDIATE owns the writer before checking this stamp again.
		// A different writer invalidates the permit and must checkpoint its tail.
		checked, err = os.Stat(s.path + "-wal")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, nil, errors.Join(err, conn.Close())
		}
	}
	return nil, nil, errors.Join(fault.New(diag.ResourceExhausted, diag.StorageWALPressure), conn.Close())
}

func (s *Store) execWrite(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tx, release, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fault.New(diag.PersistenceFailed, diag.WrappedCause, err)
	}
	return result, nil
}
