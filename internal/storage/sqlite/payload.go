// Purpose: Restore immutable payloads through their registered codec or public-room reference.
// Pattern: SQLite payload reader.
// Usage: History hydration loads selected references after closing its metadata rows.
// Related: internal/storage/sqlite/record_codec.go, internal/storage/sqlite/team_payloads.go.
package sqlite

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
)

func (s *Store) payload(ctx context.Context, id string) (json.RawMessage, error) {
	var codec, size int
	var body, digest []byte
	if _, err := payloadNumber(id); err != nil {
		return nil, err
	}
	if err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,body,digest FROM payloads WHERE id=?", id).Scan(&codec, &size, &body, &digest); err != nil {
		return nil, err
	}
	if codec == 3 {
		return s.roomPayload(ctx, hex.EncodeToString(digest), size, body)
	}
	return s.codec.Restore(codec, size, hex.EncodeToString(digest), body)
}

func payloadNumber(id string) (int64, error) {
	value, err := strconv.ParseInt(id, 10, 64)
	if err != nil || value < 1 || strconv.FormatInt(value, 10) != id {
		return 0, fmt.Errorf("invalid local payload identity")
	}
	return value, nil
}
