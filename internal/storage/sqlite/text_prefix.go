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
	id, quoted := content.Reference(id)
	value, err := s.admitPayload(ctx, id, budget)
	if err != nil {
		return "", false, err
	}
	if value.codec == 3 {
		raw, err := s.restorePayload(ctx, value.codec, value.size, value.digest, value.body)
		if err != nil {
			return "", false, err
		}
		return content.StringPrefix(raw, limit)
	}
	if quoted {
		return s.codec.RawTextPrefix(ctx, value.codec, value.size, hex.EncodeToString(value.digest), value.body, limit)
	}
	return s.codec.TextPrefix(ctx, value.codec, value.size, hex.EncodeToString(value.digest), value.body, limit)
}
