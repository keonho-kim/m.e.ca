// Purpose: Connect record content transformations to the active SQLite transaction and connection.
// Pattern: Persistence adapter for content callbacks.
// Usage: Message/event writers pack atomically; queries close their rows before restoring payloads.
// Related: internal/storage/content/pack.go, internal/storage/content/restore.go, internal/storage/sqlite/payload.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

type storedRecord struct {
	sequence   int64
	kind       string
	body, refs []byte
	size       int
}

func (s *Store) packRecord(ctx context.Context, tx *sql.Tx, body []byte) ([]byte, []byte, error) {
	return recordcontent.Pack(body, func(raw []byte) (string, error) { return s.putPayload(ctx, tx, raw) })
}

func (s *Store) unpackRecord(ctx context.Context, record storedRecord, fields map[string]bool) ([]byte, error) {
	return recordcontent.Restore(record.body, record.refs, fields, func(id string) (json.RawMessage, error) { return s.payload(ctx, id) })
}
