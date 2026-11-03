// Purpose: Verify shared public windows preserve receiver isolation, gaps, eviction and cancellation.
// Pattern: Deterministic repository-call and resource-bound tests.
// Usage: go test -race ./internal/agent/team -run TestPublicHistory.
// Related: internal/agent/team/public_history.go, internal/agent/team/inbox.go.
package team

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

type publicReader struct {
	channel.Repository
	calls  atomic.Int64
	values []channel.Message
}

func (r *publicReader) TeamMessages(_ context.Context, after, through int64, limit int) ([]channel.Message, error) {
	r.calls.Add(1)
	var out []channel.Message
	for _, value := range r.values {
		if value.Sequence > after && value.Sequence <= through && len(out) < limit {
			out = append(out, value)
		}
	}
	return out, nil
}

func TestPublicHistoryRestoresOnceAndCopiesReceiverMetadata(t *testing.T) {
	reader := &publicReader{values: []channel.Message{{Sequence: 10, From: "agent-a", Text: strings.Repeat("x", 65536)}, {Sequence: 20, From: "user", Text: "current request"}}}
	var h publicHistory
	for range 8 {
		values, err := h.read(context.Background(), reader, 0, 20, 100)
		if err != nil || len(values) != 2 || values[1].context.Content != "current request" {
			t.Fatalf("history: %d %v", len(values), err)
		}
		values[1].context.Content = "private mutation"
	}
	if calls := reader.calls.Load(); calls != 1 {
		t.Fatalf("public bodies restored %d times", calls)
	}
}

func TestPublicHistoryPublicationAndEvictionKeepCoverage(t *testing.T) {
	reader := &publicReader{}
	var h publicHistory
	for i := int64(1); i <= 130; i++ {
		m := channel.Message{Sequence: i * 2, From: "user", Text: "instruction"}
		reader.values = append(reader.values, m)
		h.append(m, (i-1)*2)
	}
	if len(h.values) != 128 || h.after != 4 || h.bytes > publicHistoryBytes {
		t.Fatal("unbounded or incorrectly indexed cache")
	}
	page, err := h.read(context.Background(), reader, 0, 260, 100)
	if err != nil || len(page) != 100 || page[0].message.Sequence != 2 || reader.calls.Load() != 1 {
		t.Fatal("evicted original was omitted")
	}
	// A new append after a partially restored older page cannot claim coverage of its gap.
	m := channel.Message{Sequence: 262, From: "user", Text: "new"}
	reader.values = append(reader.values, m)
	h.append(m, 260)
	page, err = h.read(context.Background(), reader, 200, 262, 100)
	if err != nil || len(page) != 31 || page[0].message.Sequence != 202 {
		t.Fatalf("gap in history was hidden: %d %v", len(page), err)
	}
}

func TestPublicHistoryLargeBodyAndCancellation(t *testing.T) {
	reader := &publicReader{values: []channel.Message{{Sequence: 1, From: "agent-a", Text: strings.Repeat("x", publicHistoryBytes)}}}
	var h publicHistory
	values, err := h.read(context.Background(), reader, 0, 1, 100)
	if err != nil || len(values) != 1 || len(h.values) != 0 || h.bytes != 0 {
		t.Fatal("large accepted input was rejected or retained")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.read(ctx, reader, 0, 1, 100); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled read reached the cache or repository")
	}
}

func TestPublicHistoryInboxDrainsPartialWindowThroughLatestUser(t *testing.T) {
	store := &memoryRoom{}
	ctx := context.Background()
	room := &Room{ctx: ctx, options: Options{Store: store}, latestPublic: 150}
	for i := 1; i <= 150; i++ {
		message, err := store.AppendTeamMessage(ctx, channel.Message{From: "user", Text: "latest instruction"})
		if err != nil {
			t.Fatal(err)
		}
		if i <= 50 {
			room.public.append(message, int64(i-1))
		}
	}
	peer := &Peer{ID: "agent-a", room: room}
	batch, err := peer.Inbox(ctx)
	if err != nil || len(batch.Messages) != 150 || batch.Cursor != 150 || peer.UserSequence != 150 {
		t.Fatalf("latest user was delayed behind a short cached page: %d cursor=%d %v", len(batch.Messages), batch.Cursor, err)
	}
}
