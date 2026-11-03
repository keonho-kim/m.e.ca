// Purpose: Share committed public observations across receivers within one bounded request lifetime.
// Pattern: Mutex-owned contiguous log window with repository fallback.
// Usage: Publication appends committed values; inbox and explicit history reads use read.
// Related: internal/agent/team/inbox.go, internal/agent/team/publication.go.
package team

import (
	"context"
	"slices"
	"sync"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

const publicHistoryBytes = 2 * 1024 * 1024

type publicObservation struct {
	message channel.Message
	context modelapi.Message
}

func (v publicObservation) retainedBytes() int { return len(v.context.Content) + 512 }

type publicHistory struct {
	mu             sync.Mutex
	values         []publicObservation
	after, through int64
	bytes          int
}

func observePublic(message channel.Message) publicObservation {
	value := message.Context()
	// Peer attribution and raw text share one immutable string allocation.
	message.Text = value.Content[len(value.Content)-len(message.Text):]
	return publicObservation{message, value}
}

func (h *publicHistory) append(message channel.Message, previous int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if previous == 0 {
		previous = message.Sequence - 1
	}
	if h.through != previous {
		h.reset(previous)
	}
	h.add(observePublic(message))
}

func (h *publicHistory) read(ctx context.Context, store channel.Repository, after, through int64, limit int) ([]publicObservation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if after >= h.after && after < h.through {
		first := slices.IndexFunc(h.values, func(v publicObservation) bool { return v.message.Sequence > after })
		if first >= 0 {
			end := first
			for end < len(h.values) && end-first < limit && h.values[end].message.Sequence <= through {
				end++
			}
			return slices.Clone(h.values[first:end]), nil
		}
	}
	messages, err := store.TeamMessages(ctx, after, through, limit)
	if err != nil {
		return nil, err
	}
	values := make([]publicObservation, len(messages))
	h.reset(after)
	for i, message := range messages {
		values[i] = observePublic(message)
		h.add(values[i])
	}
	return values, nil
}

func (h *publicHistory) reset(after int64) {
	clear(h.values)
	h.values = h.values[:0]
	h.after, h.through, h.bytes = after, after, 0
}

func (h *publicHistory) add(value publicObservation) {
	cost := value.retainedBytes()
	if cost > publicHistoryBytes {
		h.reset(value.message.Sequence)
		h.values = nil
		return
	}
	for len(h.values) > 0 && (h.bytes+cost > publicHistoryBytes || len(h.values) >= 128) {
		h.after = h.values[0].message.Sequence
		h.bytes -= h.values[0].retainedBytes()
		copy(h.values, h.values[1:])
		h.values[len(h.values)-1] = publicObservation{}
		h.values = h.values[:len(h.values)-1]
	}
	h.values = append(h.values, value)
	h.through = value.message.Sequence
	h.bytes += cost
}
