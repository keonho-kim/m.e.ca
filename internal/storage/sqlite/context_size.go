// Purpose: Admit selected context restoration using one bounded payload-metadata batch per page.
// Pattern: Metadata-only memory admission boundary.
// Usage: ContextMessages charges selected fields before restoring any referenced bodies.
// Related: internal/storage/sqlite/context.go, internal/storage/sqlite/read_budget.go.
package sqlite

import (
	"context"
	"encoding/json"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) contextReadSizes(ctx context.Context, page []storedRecord, fields []map[string]bool) ([]int, error) {
	sizes := make([]int, len(page))
	references := make([][]int64, len(page))
	unique := make(map[int64]bool)
	for i, record := range page {
		if fields[i] == nil {
			sizes[i] = record.size
			continue
		}
		sizes[i] = len(record.body)
		var refs map[string]string
		if err := json.Unmarshal(record.refs, &refs); err != nil {
			return nil, err
		}
		for key, id := range refs {
			var path []string
			if err := json.Unmarshal([]byte(key), &path); err != nil || len(path) == 0 {
				return nil, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadPath)
			}
			if !fields[i][path[0]] {
				continue
			}
			number, err := payloadNumber(id)
			if err != nil {
				return nil, err
			}
			references[i] = append(references[i], number)
			unique[number] = true
		}
	}
	loaded := make(map[int64]int, len(unique))
	ids := make([]any, 0, min(256, len(unique)))
	query := func() error {
		rows, err := s.db.QueryContext(ctx, "SELECT id,MAX(raw_bytes,length(body)) FROM payloads WHERE id IN ("+placeholders(len(ids))+")", ids...)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int64
			var size int
			if err := rows.Scan(&id, &size); err != nil {
				_ = rows.Close()
				return err
			}
			loaded[id] = size
		}
		err = rows.Err()
		closeErr := rows.Close()
		if err != nil {
			return err
		}
		return closeErr
	}
	for id := range unique {
		ids = append(ids, id)
		if len(ids) == 256 {
			if err := query(); err != nil {
				return nil, err
			}
			ids = ids[:0]
		}
	}
	if len(ids) > 0 {
		if err := query(); err != nil {
			return nil, err
		}
	}
	for i, refs := range references {
		for _, id := range refs {
			size, ok := loaded[id]
			if !ok || size < 1 || size > content.MaxBytes-sizes[i] {
				return nil, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadLength)
			}
			sizes[i] += size
		}
	}
	return sizes, nil
}
