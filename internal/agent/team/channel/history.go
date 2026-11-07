// Purpose: Own bounded public-channel history shared by receivers within one request lifetime.
// Pattern: Mutex-owned contiguous log window with repository fallback.
// Usage: Publication appends committed values; inbox and explicit history reads use read.
// Related: internal/agent/team/inbox.go, internal/agent/team/publication.go.
package channel

import (
	"slices"
	"sync"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

const historyBytes = 2 * 1024 * 1024

type Entry struct {
	Message Message
	Context modelapi.Message
}

func (v Entry) retainedBytes() int {
	return len(v.Context.Content) + 512 + v.Message.metadataBytes()
}

func (m Message) metadataBytes() int {
	size := 0
	if note := m.Note; note != nil {
		for _, name := range note.Mentions {
			size += len(name) + 16
		}
		if work := note.Work; work != nil {
			size += len(work.Contract)
			for _, path := range work.Paths {
				size += len(path) + 16
			}
		}
	}
	return size
}

type History struct {
	mu             sync.Mutex
	loading        *historyLoad
	values         []Entry
	after, through int64
	bytes          int
}

// Metadata borrows only immutable identity and sizing facts from the current window.
func (h *History) Metadata(sequence int64) (Metadata, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, value := range h.values {
		if value.Message.Sequence == sequence {
			m := value.Message
			return Metadata{Sequence: sequence, From: m.From, Kind: m.Kind(), Bytes: m.ContextBytes()}, true
		}
	}
	return Metadata{}, false
}

func observePublic(message Message) Entry {
	value := message.Context()
	// Peer attribution and raw text share one immutable string allocation.
	boundary := len(value.Content) - len(message.Text)
	prefix := value.Content[:boundary]
	message.Prefix = &prefix
	message.Text = value.Content[boundary:]
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
	// Cache admission precedes the potentially large attribution allocation.
	if message.ContextBytes()+message.metadataBytes() > historyBytes {
		h.reset(message.Sequence)
		h.values = nil
		return
	}
	h.add(observePublic(message))
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
