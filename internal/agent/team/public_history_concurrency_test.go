// Purpose: Preserve the latest public window and publication while older records load.
// Pattern: Deterministic cache and repository concurrency tests.
// Usage: Team tests check historical reads without model or terminal I/O.
// Related: internal/agent/team/public_history.go, internal/agent/team/public_history_test.go.
package team

import (
	"context"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestPublicHistoryOlderReadPreservesLatestWindow(t *testing.T) {
	reader := &publicReader{values: []channel.Message{{Sequence: 1, From: "user", Text: "old"}, {Sequence: 2, From: "user", Text: "latest"}}}
	var h publicHistory
	h.append(reader.values[1], 1)
	if _, err := h.read(context.Background(), reader, 0, 1, 100); err != nil {
		t.Fatal(err)
	}
	if values, err := h.read(context.Background(), reader, 1, 2, 100); err != nil || len(values) != 1 || values[0].message.Text != "latest" {
		t.Fatal("latest window lost", err)
	}
	if reader.calls.Load() != 1 {
		t.Fatal("historical lookup evicted the latest message")
	}
}

type blockedPublicReader struct {
	channel.Repository
	entered, release chan struct{}
}

func (r blockedPublicReader) TeamMessages(context.Context, int64, int64, int) ([]channel.Message, error) {
	close(r.entered)
	<-r.release
	return []channel.Message{{Sequence: 1, From: "user", Text: "old"}}, nil
}

func TestPublicHistoryPublicationDoesNotWaitForHistoricalIO(t *testing.T) {
	reader := blockedPublicReader{entered: make(chan struct{}), release: make(chan struct{})}
	var h publicHistory
	loaded := make(chan error, 1)
	go func() { _, err := h.read(context.Background(), reader, 0, 1, 100); loaded <- err }()
	<-reader.entered
	published := make(chan struct{})
	go func() { h.append(channel.Message{Sequence: 2, From: "user", Text: "new"}, 1); close(published) }()
	blocked := false
	select {
	case <-published:
	case <-time.After(2 * time.Second):
		blocked = true
	}
	close(reader.release)
	if err := <-loaded; err != nil {
		t.Fatal(err)
	}
	<-published
	if blocked {
		t.Fatal("publication waited for historical I/O")
	}
	if h.through != 2 {
		t.Fatal("old read replaced concurrently published context")
	}
}
