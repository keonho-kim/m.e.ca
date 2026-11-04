// Purpose: Share concurrent cold public-page reads without retaining oversized results afterward.
// Pattern: One request-owned in-flight read with independently cancellable waiters.
// Usage: History.Read joins identical reads; publication and cached reads remain independent.
// Related: internal/agent/team/channel/history.go, internal/agent/team/channel/message.go.
package channel

import (
	"context"
	"errors"
	"slices"
)

type historyLoad struct {
	after, through int64
	limit, waiters int
	done           chan struct{}
	values         []Entry
	err            error
}

func (h *History) Read(ctx context.Context, store Reader, after, through int64, limit int) ([]Entry, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		h.mu.Lock()
		if values, ok := h.cached(after, through, limit); ok {
			h.mu.Unlock()
			return values, nil
		}
		if pending := h.loading; pending != nil {
			same := pending.after == after && pending.through == through && pending.limit == limit
			if same {
				pending.waiters++
			}
			h.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-pending.done:
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if same {
				// A cancelled original caller does not cancel another receiver's own read.
				if errors.Is(pending.err, context.Canceled) || errors.Is(pending.err, context.DeadlineExceeded) {
					continue
				}
				return slices.Clone(pending.values), pending.err
			}
			continue
		}
		pending := &historyLoad{after: after, through: through, limit: limit, done: make(chan struct{})}
		oldAfter, oldThrough := h.after, h.through
		h.loading = pending
		h.mu.Unlock()
		messages, err := store.TeamMessages(ctx, after, through, limit)
		var values []Entry
		if err == nil {
			values = make([]Entry, len(messages))
			for i, message := range messages {
				values[i] = observePublic(message)
			}
		}
		h.mu.Lock()
		if err == nil && h.after == oldAfter && h.through == oldThrough && (h.through == 0 || after == h.through) {
			if len(h.values) == 0 {
				h.reset(after)
			}
			for _, value := range values {
				h.add(value)
			}
		}
		pending.values, pending.err = values, err
		h.loading = nil
		close(pending.done)
		h.mu.Unlock()
		if pending.waiters > 0 {
			return slices.Clone(values), err
		}
		return values, err
	}
}
