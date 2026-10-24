// Purpose: Read small stored-record pages before releasing the connection for payload hydration.
// Pattern: SQLite projection boundary.
// Usage: Timeline, context, tail selection and export read through recordPage.
// Related: internal/storage/sqlite/record_codec.go, internal/storage/sqlite/timeline.go.
package sqlite

import (
	"context"
	"errors"
	"fmt"
)

func (s *Store) recordPage(ctx context.Context, query string, args ...any) (records []storedRecord, err error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var r storedRecord
		if err := rows.Scan(&r.sequence, &r.kind, &r.body, &r.refs, &r.size); err != nil {
			return nil, err
		}
		if r.size < 1 || r.size > maxEventBytes {
			return nil, fmt.Errorf("invalid stored record length")
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
