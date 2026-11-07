// Purpose: Reject oversized selected payloads before allocating their restored bodies.
// Pattern: Operation-owned restoration budget.
// Usage: Timeline, tool detail and compatible output readers share one budget per operation.
// Related: internal/storage/sqlite/payload.go, internal/storage/history/read.go.
package sqlite

import (
	"context"
	"encoding/json"

	"github.com/keonho-kim/meca/internal/storage/history"
)

type readBudget struct {
	remaining int
	limited   bool
}

func newReadBudget(maxBytes int) *readBudget {
	return &readBudget{remaining: maxBytes, limited: maxBytes > 0}
}

func (b *readBudget) take(size int) error {
	if b.limited {
		if size < 0 || size > b.remaining {
			return history.ErrReadLimit
		}
		b.remaining -= size
	}
	return nil
}

func (s *Store) readPayload(ctx context.Context, id string, budget *readBudget) (json.RawMessage, error) {
	if _, err := payloadNumber(id); err != nil {
		return nil, err
	}
	if budget.limited {
		var size, stored int
		if err := s.db.QueryRowContext(ctx, "SELECT raw_bytes,length(body) FROM payloads WHERE id=?", id).Scan(&size, &stored); err != nil {
			return nil, err
		}
		if err := budget.take(max(size, stored)); err != nil {
			return nil, err
		}
	}
	return s.payload(ctx, id)
}
