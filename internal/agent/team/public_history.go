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
	loadMu         sync.Mutex
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
	values, ok := h.cached(after, through, limit)
	h.mu.Unlock()
	if ok {
		return values, nil
	}
	// Serialize cold loads, but leave recent reads and publication independent
	// of repository latency. Recheck after another receiver finishes loading.
	h.loadMu.Lock()
	defer h.loadMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	h.mu.Lock()
	values, ok = h.cached(after, through, limit)
	oldAfter, oldThrough := h.after, h.through
	h.mu.Unlock()
	if ok {
		return values, nil
	}
	messages, err := store.TeamMessages(ctx, after, through, limit)
	if err != nil {
		return nil, err
	}
	values = make([]publicObservation, len(messages))
	for i, message := range messages {
		values[i] = observePublic(message)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	// Historical pages never displace a newer window or a concurrent append.
	if h.after == oldAfter && h.through == oldThrough && (h.through == 0 || after == h.through) {
		if len(h.values) == 0 {
			h.reset(after)
		}
		for _, value := range values {
			h.add(value)
		}
	}
	return values, nil
}

// cached requires mu and copies only receiver-owned message headers.
func (h *publicHistory) cached(after, through int64, limit int) ([]publicObservation, bool) {
	if after >= h.after && after < h.through {
		first := slices.IndexFunc(h.values, func(v publicObservation) bool { return v.message.Sequence > after })
		if first >= 0 {
			end := first
			for end < len(h.values) && end-first < limit && h.values[end].message.Sequence <= through {
				end++
			}
			return slices.Clone(h.values[first:end]), true
		}
	}
	return nil, false
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
