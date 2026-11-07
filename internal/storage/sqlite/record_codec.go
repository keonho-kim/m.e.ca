// Purpose: Restore selected record fields from immutable payload references.
// Pattern: Persistence adapter for content read callbacks.
// Usage: History queries close metadata rows before restoring their selected fields.
// Related: internal/storage/content/restore.go, internal/storage/sqlite/payload.go.
package sqlite

import (
	"context"
	"encoding/json"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

type storedRecord struct {
	sequence   int64
	kind       string
	body, refs []byte
	size       int
}

func (s *Store) unpackRecord(ctx context.Context, record storedRecord, fields map[string]bool) ([]byte, error) {
	return recordcontent.Restore(record.body, record.refs, fields, func(id string) (json.RawMessage, error) { return s.payload(ctx, id) })
}
