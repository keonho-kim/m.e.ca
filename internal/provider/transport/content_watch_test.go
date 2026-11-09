// Purpose: Verify keepalives cannot extend generation stalls and consumers do not spend network wait time.
// Pattern: Watchdog lifecycle tests with virtual time.
// Usage: go test -race ./internal/provider/transport -run TestContentWatch.
// Related: internal/provider/transport/content_watch.go.
package transport

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

func TestContentWatchRequiresMeaningfulProgress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		watch := WatchContent(t.Context(), time.Second)
		defer watch.Close()
		for range 4 {
			time.Sleep(200 * time.Millisecond)
		}
		if watch.Context().Err() != nil {
			t.Fatal("watch expired early")
		}
		watch.Activity()
		time.Sleep(900 * time.Millisecond)
		if watch.Context().Err() != nil {
			t.Fatal("real progress did not extend the window")
		}
		time.Sleep(101 * time.Millisecond)
		synctest.Wait()
		if context.Cause(watch.Context()) == nil {
			t.Fatal("connection without content progress stayed alive")
		}
	})
}

func TestContentWatchPausesOwnedConsumerWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		watch := WatchContent(t.Context(), time.Second)
		defer watch.Close()
		time.Sleep(500 * time.Millisecond)
		if err := watch.Consume(func() error { time.Sleep(10 * time.Second); return nil }); err != nil {
			t.Fatal(err)
		}
		if watch.Context().Err() != nil {
			t.Fatal("consumer work consumed provider wait budget")
		}
		time.Sleep(501 * time.Millisecond)
		synctest.Wait()
		if watch.Context().Err() == nil {
			t.Fatal("pause reset a non-progress event's whole window")
		}
	})
}
