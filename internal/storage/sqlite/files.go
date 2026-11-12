// Purpose: Persist explicit file and directory references without collecting their contents.
// Pattern: Repository.
// Usage: Used by file selection and session restoration.
// Related: internal/tui/session/files.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"context"
	"errors"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) FileSelections(ctx context.Context, session string) (out []string, err error) {
	if err := s.checkSession(session); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT path FROM file_selections ORDER BY path")
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var path string
		if err = rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}
func (s *Store) SelectFile(ctx context.Context, session, path string, selected bool) error {
	if err := s.checkSession(session); err != nil {
		return err
	}
	if len(path) > 4096 {
		return faults.Errorf(diagcatalog.SqliteInvalidSelectedPath)
	}
	if selected {
		_, err := s.execWrite(ctx, "INSERT OR IGNORE INTO file_selections(path) VALUES(?)", path)
		return err
	}
	_, err := s.execWrite(ctx, "DELETE FROM file_selections WHERE path=?", path)
	return err
}
