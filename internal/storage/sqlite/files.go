// Purpose: Persist explicit file and directory references without collecting their contents.
// Pattern: Repository.
// Usage: Used by file selection and session restoration.
// Related: internal/tui/session/files.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"context"
	"errors"
	"fmt"
)

func (s *Store) FileSelections(ctx context.Context, session string) (out []string, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT path FROM file_selections WHERE session_id=? ORDER BY path", session)
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
	if len(path) > 4096 {
		return fmt.Errorf("invalid selected path")
	}
	if selected {
		_, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO file_selections(session_id,path) VALUES(?,?)", session, path)
		return err
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM file_selections WHERE session_id=? AND path=?", session, path)
	return err
}
