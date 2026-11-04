// Purpose: Verify oversized in-flight sharing, cancellation isolation and allocation-free cache rejection.
// Pattern: Controlled concurrent repository tests and allocation benchmark.
// Usage: go test -race ./internal/agent/team/channel.
// Related: internal/agent/team/channel/history_load.go, internal/agent/team/channel/history.go.
package channel

import (
	"context"
	record "github.com/keonho-kim/meca/internal/agent/record"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type sharedLargeReader struct {
	calls            atomic.Int64
	entered, release chan struct{}
	body             string
}

func (r *sharedLargeReader) TeamMessages(ctx context.Context, _, _ int64, _ int) ([]Message, error) {
	if r.calls.Add(1) == 1 {
		close(r.entered)
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-r.release:
	}
	return []Message{{Sequence: 1, From: "Hana", Text: r.body}}, nil
}

func TestOversizedConcurrentReadSharesOnlyItsInflightBody(t *testing.T) {
	r := &sharedLargeReader{entered: make(chan struct{}), release: make(chan struct{}), body: strings.Repeat("x", historyBytes)}
	var h History
	type result struct {
		values []Entry
		err    error
	}
	out := make(chan result, 8)
	for range 8 {
		go func() { v, e := h.Read(context.Background(), r, 0, 1, 100); out <- result{v, e} }()
	}
	<-r.entered
	deadline := time.Now().Add(3 * time.Second)
	for {
		h.mu.Lock()
		ready := h.loading != nil && h.loading.waiters == 7
		h.mu.Unlock()
		if ready {
			break
		}
		if time.Now().After(deadline) {
			close(r.release)
			t.Fatal("receivers failed to join the in-flight read")
		}
		runtime.Gosched()
	}
	close(r.release)
	for range 8 {
		v := <-out
		if v.err != nil || len(v.values) != 1 {
			t.Fatal(v.err)
		}
	}
	if r.calls.Load() != 1 || h.loading != nil || len(h.values) != 0 {
		t.Fatal("oversized body was copied or retained", r.calls.Load())
	}
	if _, err := h.Read(context.Background(), r, 0, 1, 100); err != nil || r.calls.Load() != 2 {
		t.Fatal("completed oversized load was retained", err)
	}
}

func TestCancelledWaiterDoesNotCancelSharedRead(t *testing.T) {
	r := &sharedLargeReader{entered: make(chan struct{}), release: make(chan struct{}), body: "text"}
	var h History
	owner := make(chan error, 1)
	go func() { _, err := h.Read(context.Background(), r, 0, 1, 100); owner <- err }()
	<-r.entered
	ctx, cancel := context.WithCancel(context.Background())
	waiter := make(chan error, 1)
	go func() { _, err := h.Read(ctx, r, 0, 1, 100); waiter <- err }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		h.mu.Lock()
		joined := h.loading != nil && h.loading.waiters == 1
		h.mu.Unlock()
		if joined {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			close(r.release)
			t.Fatal("waiter did not join")
		}
		runtime.Gosched()
	}
	cancel()
	if err := <-waiter; err != context.Canceled {
		t.Fatal(err)
	}
	close(r.release)
	if err := <-owner; err != nil || r.calls.Load() != 1 {
		t.Fatal("waiter cancelled the original read", err)
	}
}

func TestCancelledOriginalReaderLetsLiveWaiterReload(t *testing.T) {
	r := &sharedLargeReader{entered: make(chan struct{}), release: make(chan struct{}), body: "text"}
	var h History
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner, waiter := make(chan error, 1), make(chan error, 1)
	go func() { _, err := h.Read(ctx, r, 0, 1, 100); owner <- err }()
	<-r.entered
	go func() { _, err := h.Read(context.Background(), r, 0, 1, 100); waiter <- err }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		h.mu.Lock()
		joined := h.loading != nil && h.loading.waiters == 1
		h.mu.Unlock()
		if joined {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			close(r.release)
			t.Fatal("waiter did not join")
		}
		runtime.Gosched()
	}
	cancel()
	if err := <-owner; err != context.Canceled {
		t.Fatal(err)
	}
	close(r.release)
	if err := <-waiter; err != nil || r.calls.Load() != 2 {
		t.Fatal("cancelled original poisoned a live receiver", err)
	}
}

func BenchmarkOversizedPublicAppend(b *testing.B) {
	for _, m := range []Message{
		{Sequence: 1, From: "Hana", Text: strings.Repeat("x", 4<<20)},
		{Sequence: 1, From: "Hana", Text: "Work", Note: &record.TeamNote{Work: &record.TeamWork{Contract: strings.Repeat("x", 1500<<10), Revision: 1}}},
	} {
		b.Run(m.Text[:4], func(b *testing.B) {
			var h History
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				h.Append(m, 0)
			}
		})
	}
}
