// Purpose: Reject oversized selected payloads before allocating their restored bodies.
// Pattern: Operation-owned restoration budget.
// Usage: Timeline, tool detail and compatible output readers share one budget per operation.
// Related: internal/storage/sqlite/payload.go, internal/storage/history/read.go.
package sqlite

import (
	"context"
	"encoding/json"

	"github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"
)

type readBudget struct {
	remaining int
	limited   bool
	selected  int
	reserve   func(int) error
}

func newReadBudget(maxBytes int, reserve func(int) error) *readBudget {
	return &readBudget{remaining: maxBytes, limited: maxBytes > 0, reserve: reserve}
}

func (b *readBudget) take(size int) error {
	if b.limited {
		if size < 0 || size > b.remaining {
			return history.ErrReadLimit
		}
	}
	if b.reserve != nil {
		if err := b.reserve(b.selected + size); err != nil {
			return err
		}
		b.selected += size
	}
	if b.limited {
		b.remaining -= size
	}
	return nil
}

func (s *Store) readPayload(ctx context.Context, id string, budget *readBudget) (json.RawMessage, error) {
	id, _ = content.Reference(id)
	if budget.reserve != nil {
		value, err := s.admitPayload(ctx, id, budget)
		if err != nil {
			return nil, err
		}
		return s.restorePayload(ctx, value.codec, value.size, value.digest, value.body)
	}
	if _, err := payloadNumber(id); err != nil {
		return nil, err
	}
	if !budget.limited {
		return s.payload(ctx, id)
	}
	var codec, size, stored int
	var body, digest []byte
	err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,length(body),digest,CASE WHEN MAX(raw_bytes,length(body))<=? THEN body END FROM payloads WHERE id=?", budget.remaining, id).Scan(&codec, &size, &stored, &digest, &body)
	if err != nil {
		return nil, err
	}
	if err := budget.take(max(size, stored)); err != nil {
		return nil, err
	}
	return s.restorePayload(ctx, codec, size, digest, body)
}
