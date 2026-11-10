// Purpose: Verify shared thumbnail work has one producer and canceled readers release their wait.
// Pattern: Deterministic concurrency contract test using explicit barriers.
// Usage: go test -race ./internal/serverapi -run TestThumbnailCache.
// Related: internal/serverapi/thumbnail.go.
package serverapi

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestThumbnailCacheCoalescesAndCancelsWaiters(t *testing.T) {
	var cache thumbnailCache
	var calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	result := make(chan error, 4)
	load := func() ([]byte, error) {
		calls.Add(1)
		close(entered)
		<-release
		return []byte("immutable preview"), nil
	}
	go func() { _, err := cache.get(t.Context(), "same-source", load); result <- err }()
	<-entered
	ctx, cancel := context.WithCancel(t.Context())
	waiting := make(chan struct{})
	go func() { close(waiting); _, err := cache.get(ctx, "same-source", load); result <- err }()
	<-waiting
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	for range 2 {
		go func() { _, err := cache.get(t.Context(), "same-source", load); result <- err }()
	}
	once.Do(func() { close(release) })
	for range 3 {
		if err := <-result; err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("same immutable source decoded repeatedly", calls.Load())
	}
	if len(cache.pending) != 0 {
		t.Fatal("completed producer retained waiters")
	}
}
