// Purpose: Restore immutable payloads through their registered codec or public-room reference.
// Pattern: SQLite payload reader.
// Usage: History hydration loads selected references after closing its metadata rows.
// Related: internal/storage/sqlite/record_codec.go, internal/storage/sqlite/team_payloads.go.
package sqlite

import (
	"context"
	"encoding/hex"
	"encoding/json"

	"strconv"

	"github.com/keonho-kim/meca/internal/storage/content"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) payload(ctx context.Context, id string) (json.RawMessage, error) {
	id, _ = content.Reference(id)
	var codec, size int
	var body, digest []byte
	if _, err := payloadNumber(id); err != nil {
		return nil, err
	}
	if err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,body,digest FROM payloads WHERE id=?", id).Scan(&codec, &size, &body, &digest); err != nil {
		return nil, err
	}
	return s.restorePayload(ctx, codec, size, digest, body)
}

func (s *Store) restorePayload(ctx context.Context, codec, size int, digest, body []byte) (json.RawMessage, error) {
	if codec == 3 {
		return s.roomPayload(ctx, hex.EncodeToString(digest), size, body)
	}
	return s.codec.Restore(codec, size, hex.EncodeToString(digest), body)
}

func payloadNumber(id string) (int64, error) {
	id, _ = content.Reference(id)
	value, err := strconv.ParseInt(id, 10, 64)
	if err != nil || value < 1 || strconv.FormatInt(value, 10) != id {
		return 0, faults.Errorf(diagcatalog.SqliteInvalidLocalPayloadIdentity)
	}
	return value, nil
}
