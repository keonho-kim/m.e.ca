// Purpose: Store immutable large JSON values once with compression and integrity checks.
// Pattern: SQLite payload adapter.
// Usage: Record encoding writes through its transaction; hydration loads through the session DB.
// Related: internal/storage/sqlite/record_codec.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const payloadThreshold = 32 * 1024

func (s *Store) putPayload(ctx context.Context, tx *sql.Tx, raw []byte) (string, error) {
	digest := sha256.Sum256(raw)
	id := hex.EncodeToString(digest[:])
	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM payloads WHERE id=?", id).Scan(&exists); err != nil {
		return "", err
	}
	if exists > 0 {
		return id, nil
	}
	compressed, err := s.codec.encode(raw)
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
	if size < 1 || size > maxEventBytes {
		return nil, fmt.Errorf("invalid stored payload length")
	}
	switch codec {
	case 0:
	case 1:
		r, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		decoded, readErr := io.ReadAll(io.LimitReader(r, int64(size)+1))
		if err := errors.Join(readErr, r.Close()); err != nil {
			return nil, err
		}
		body = decoded
	case 2:
		decoded, err := s.codec.decode(body, size)
		if err != nil {
			return nil, err
		}
		body = decoded
	default:
		return nil, fmt.Errorf("unsupported payload codec %d", codec)
	}
	digest := sha256.Sum256(body)
	if len(body) != size || hex.EncodeToString(digest[:]) != id || !json.Valid(body) {
		return nil, fmt.Errorf("stored payload integrity check failed")
	}
	return body, nil
}
