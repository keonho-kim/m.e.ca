// Purpose: Own bounded public-channel history shared by receivers within one request lifetime.
// Pattern: Mutex-owned contiguous log window with repository fallback.
// Usage: Publication appends committed values; inbox and explicit history reads use read.
// Related: internal/agent/team/inbox.go, internal/agent/team/publication.go.
package channel

import (
	"context"
	"slices"
	"sync"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

const historyBytes = 2 * 1024 * 1024

type Entry struct {
	Message Message
	Context modelapi.Message
}

func (v Entry) retainedBytes() int { return len(v.Context.Content) + 512 }

type History struct {
	mu             sync.Mutex
	loadMu         sync.Mutex
	values         []Entry
	after, through int64
	bytes          int
}

func observePublic(message Message) Entry {
	value := message.Context()
	// Peer attribution and raw text share one immutable string allocation.
	message.Text = value.Content[len(value.Content)-len(message.Text):]
	return Entry{message, value}
}

func (h *History) Append(message Message, previous int64) {
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

func (h *History) Read(ctx context.Context, store Reader, after, through int64, limit int) ([]Entry, error) {
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
	values = make([]Entry, len(messages))
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
func (h *History) cached(after, through int64, limit int) ([]Entry, bool) {
	if after >= h.after && after < h.through {
		first := slices.IndexFunc(h.values, func(v Entry) bool { return v.Message.Sequence > after })
		if first >= 0 {
			end := first
			for end < len(h.values) && end-first < limit && h.values[end].Message.Sequence <= through {
				end++
			}
			return slices.Clone(h.values[first:end]), true
		}
	}
	return nil, false
}

func (h *History) reset(after int64) {
	clear(h.values)
	h.values = h.values[:0]
	h.after, h.through, h.bytes = after, after, 0
}

func (h *History) add(value Entry) {
	cost := value.retainedBytes()
	if cost > historyBytes {
		h.reset(value.Message.Sequence)
		h.values = nil
		return
	}
	for len(h.values) > 0 && (h.bytes+cost > historyBytes || len(h.values) >= 128) {
		h.after = h.values[0].Message.Sequence
		h.bytes -= h.values[0].retainedBytes()
		copy(h.values, h.values[1:])
		h.values[len(h.values)-1] = Entry{}
		h.values = h.values[:len(h.values)-1]
	}
	h.values = append(h.values, value)
	h.through = value.Message.Sequence
	h.bytes += cost
}
