// Purpose: Send heartbeats during slow I/O and join the worker when its observer disconnects.
// Pattern: Fake-clock and channel-controlled streaming tests.
// Usage: go test -race ./internal/serverapi/eventstream -run TestAwait.
// Related: internal/serverapi/eventstream/await.go, internal/serverapi/eventstream/idle.go.
package eventstream

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

type heartbeatRecorder struct {
	mu       sync.Mutex
	recorder *httptest.ResponseRecorder
}

func (r *heartbeatRecorder) Header() http.Header { return r.recorder.Header() }
func (r *heartbeatRecorder) WriteHeader(code int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recorder.WriteHeader(code)
}
func (r *heartbeatRecorder) Write(body []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recorder.Write(body)
}
func (r *heartbeatRecorder) Flush() { r.mu.Lock(); defer r.mu.Unlock(); r.recorder.Flush() }
func (r *heartbeatRecorder) body() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recorder.Body.String()
}

func TestAwaitHeartbeatsAndJoinsCanceledIO(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		recorder := &heartbeatRecorder{recorder: httptest.NewRecorder()}
		writer, err := New(recorder)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		joined := make(chan struct{})
		finished := make(chan error, 1)
		go func() {
			_, err := Await(ctx, writer, func(ctx context.Context) (int, error) {
				defer close(joined)
				<-ctx.Done()
				return 0, ctx.Err()
			})
			finished <- err
		}()
		synctest.Wait()
		time.Sleep(4 * time.Second)
		synctest.Wait()
		if recorder.body() != "" {
			t.Fatal("early heartbeat")
		}
		for want := 1; want <= 3; want++ {
			time.Sleep(time.Second)
			synctest.Wait()
			if got := strings.Count(recorder.body(), ": heartbeat\n\n"); got != want {
				t.Fatal(got, want)
			}
		}
		cancel()
		if err := <-finished; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		select {
		case <-joined:
		default:
			t.Fatal("I/O worker outlived its observer")
		}
	})
}
