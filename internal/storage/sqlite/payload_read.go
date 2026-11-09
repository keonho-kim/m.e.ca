// Purpose: Admit immutable payload metadata before selecting its encoded body.
// Pattern: Two-stage SQLite read boundary.
// Usage: Bounded text previews and API original reservations share this read path.
// Related: internal/storage/sqlite/read_budget.go, internal/storage/sqlite/text_prefix.go, internal/storage/sqlite/payload.go.
package sqlite

import "context"

type admittedPayload struct {
	codec, size  int
	digest, body []byte
}

func (s *Store) admitPayload(ctx context.Context, id string, budget *readBudget) (admittedPayload, error) {
	var value admittedPayload
	if _, err := payloadNumber(id); err != nil {
		return value, err
	}
	var stored int
	if err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,length(body),digest FROM payloads WHERE id=?", id).Scan(&value.codec, &value.size, &stored, &value.digest); err != nil {
		return value, err
	}
	if err := budget.take(max(value.size, stored)); err != nil {
		return value, err
	}
	// An append-only payload cannot change between these reads. Matching its
	// metadata also rejects a replacement before allocating the selected body.
	err := s.db.QueryRowContext(ctx, "SELECT body FROM payloads WHERE id=? AND codec=? AND raw_bytes=? AND length(body)=? AND digest=?", id, value.codec, value.size, stored, value.digest).Scan(&value.body)
	return value, err
}
