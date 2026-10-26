// Purpose: Store immutable large JSON values once with compression and integrity checks.
// Pattern: SQLite payload adapter.
// Usage: Record encoding writes through its transaction; hydration loads through the session DB.
// Related: internal/storage/sqlite/record_codec.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) putPayload(ctx context.Context, tx *sql.Tx, raw []byte) (string, error) {
	id := recordcontent.ID(raw)
	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM payloads WHERE id=?", id).Scan(&exists); err != nil {
		return "", err
	}
	if exists > 0 {
		return id, nil
	}
	compressed, err := s.codec.Encode(raw)
	if err != nil {
		return "", err
	}
	codec, body := 0, raw
	if len(compressed) < len(raw) {
		codec, body = 2, compressed
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO payloads(id,codec,raw_bytes,body) VALUES(?,?,?,?)", id, codec, len(raw), body)
	return id, err
}

func (s *Store) payload(ctx context.Context, id string) (json.RawMessage, error) {
	var codec, size int
	var body []byte
	if err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,body FROM payloads WHERE id=?", id).Scan(&codec, &size, &body); err != nil {
		return nil, err
	}
	return s.codec.Restore(codec, size, id, body)
}
