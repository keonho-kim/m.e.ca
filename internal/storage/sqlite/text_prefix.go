// Purpose: Read authenticated text previews without restoring an entire compressed string.
// Pattern: SQLite adapter for bounded content decoding.
// Usage: TimelineSummaries charges raw bytes before reading the encoded payload.
// Related: internal/storage/content/prefix_reader.go, internal/storage/sqlite/read_budget.go.
package sqlite

import (
	"context"
	"encoding/hex"
	"github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) readTextPrefix(ctx context.Context, id string, budget *readBudget, limit int) (string, bool, error) {
	if _, err := payloadNumber(id); err != nil {
		return "", false, err
	}
	var codec, size, stored int
	var digest []byte
	if err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,length(body),digest FROM payloads WHERE id=?", id).Scan(&codec, &size, &stored, &digest); err != nil {
		return "", false, err
	}
	if err := budget.take(max(size, stored)); err != nil {
		return "", false, err
	}
	if codec == 3 {
		raw, err := s.payload(ctx, id)
		if err != nil {
			return "", false, err
		}
		return content.StringPrefix(raw, limit)
	}
	var body []byte
	if err := s.db.QueryRowContext(ctx, "SELECT body FROM payloads WHERE id=? AND codec=? AND raw_bytes=? AND length(body)=?", id, codec, size, stored).Scan(&body); err != nil {
		return "", false, err
	}
	return s.codec.TextPrefix(ctx, codec, size, hex.EncodeToString(digest), body, limit)
}
