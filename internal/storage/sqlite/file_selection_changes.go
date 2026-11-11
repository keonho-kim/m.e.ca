// Purpose: Commit file-reference replacements together after a workspace rename, move or deletion.
// Pattern: SQLite transaction adapter.
// Usage: Server entry mutations submit exact old-to-new paths and receive the authoritative selection.
// Related: internal/commands/server/file_mutation.go, internal/storage/sqlite/files.go.
package sqlite

import (
	"context"
	"errors"
	"slices"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) ChangeFileSelections(ctx context.Context, session string, changes map[string]string) (out []string, resultErr error) {
	if err := s.checkSession(session); err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return s.FileSelections(ctx, session)
	}
	for from, to := range changes {
		if from == "" || len(from) > 4096 || len(to) > 4096 {
			return nil, fault.Errorf(diag.SqliteInvalidSelectedPath)
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, tx.Rollback())
		}
	}()
	rows, err := tx.QueryContext(ctx, "SELECT path FROM file_selections ORDER BY path")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, errors.Join(err, rows.Close())
		}
		out = append(out, path)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, err
	}
	// Read the original set before any write so overlapping replacements and
	// duplicate destinations cannot cascade through paths added by this batch.
	for _, path := range out {
		if next, changed := changes[path]; changed && next != path {
			if _, err := tx.ExecContext(ctx, "DELETE FROM file_selections WHERE path=?", path); err != nil {
				return nil, err
			}
		}
	}
	for i, path := range out {
		if next, changed := changes[path]; changed && next != path {
			out[i] = next
			if next != "" {
				if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO file_selections(path) VALUES(?)", next); err != nil {
					return nil, err
				}
			}
		}
	}
	slices.Sort(out)
	out = slices.Compact(out)
	if len(out) > 0 && out[0] == "" {
		out = out[1:]
	}
	return out, tx.Commit()
}
