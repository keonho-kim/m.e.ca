// Purpose: Verify native socket and RPC subscriptions emit heartbeat without advancing event cursors.
// Pattern: Fake-clock subscription contract test.
// Usage: go test ./internal/serverapi -run TestNativeHeartbeat.
// Related: internal/serverapi/watch.go, internal/serverapi/eventstream/idle.go.
package serverapi

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestNativeHeartbeatAfterFiveSecondsThenEverySecond(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	cursor := f.app.State().Cursor
	synctest.Test(t, func(t *testing.T) {
		defer func() {
			if err := f.app.Close(); err != nil {
				t.Error(err)
			}
		}()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		frames := make(chan *v1.Notification, 8)
		done := make(chan error, 1)
		go func() {
			done <- f.api.watch(ctx, cursor, "", func(n *v1.Notification) error { frames <- n; return nil })
		}()
		synctest.Wait()
		time.Sleep(4999 * time.Millisecond)
		select {
		case n := <-frames:
			t.Fatal("early heartbeat", n)
		default:
		}
		for _, delay := range []time.Duration{time.Millisecond, time.Second, time.Second} {
			time.Sleep(delay)
			synctest.Wait()
			select {
			case n := <-frames:
				if n.GetEvent().Kind != "heartbeat" || n.GetEvent().Cursor != "" {
					t.Fatal(n)
				}
			default:
				t.Fatal("heartbeat missing")
			}
		}
		cancel()
		synctest.Wait()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if f.app.State().Cursor != cursor {
			t.Fatal("heartbeat entered replay log")
		}
	})
}
