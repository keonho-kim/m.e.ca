// Purpose: Verify only stalled I/O expires, with deterministic long-running stream and recovery tests.
// Pattern: Transport lifecycle tests with virtual time.
// Usage: go test ./internal/provider/httpclient -run TestIdle.
// Related: internal/provider/transport/idle.go, internal/provider/transport/retry.go.
package httpclient

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type pacedBody struct {
	ctx       context.Context
	remaining int
	delay     time.Duration
}

func (b *pacedBody) Read(p []byte) (int, error) {
	if b.remaining < 0 {
		return 0, io.EOF
	}
	timer := time.NewTimer(b.delay)
	defer timer.Stop()
	select {
	case <-b.ctx.Done():
		return 0, b.ctx.Err()
	case <-timer.C:
	}
	frame := "data: {\"choices\":[{\"delta\":{\"content\":\"x\"}}]}\n\n"
	if b.remaining == 0 {
		frame = "data: [DONE]\n\n"
	}
	b.remaining--
	return copy(p, frame), nil
}
func (*pacedBody) Close() error { return nil }

func TestIdleLongStreamAndRecovery(t *testing.T) {
	for _, stalled := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			conn, err := connection.New(connection.VLLM, "http://localhost:8000")
			if err != nil {
				t.Fatal(err)
			}
			client, err := NewClient(conn, "")
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			attempts := 0
			client.transport.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				attempts++
				delay := 30 * time.Second
				if stalled && attempts == 1 {
					delay = 2 * time.Minute
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: &pacedBody{ctx: r.Context(), remaining: 10, delay: delay}}, nil
			})
			request := recoveryRequest()
			request.IdleTimeout = time.Minute
			var answer strings.Builder
			err = client.Generate(context.Background(), request, func(e modelapi.Event) error {
				answer.WriteString(e.Delta)
				if e.Delta != "" {
					time.Sleep(2 * time.Minute)
				} // Virtual consumer backpressure must not trigger network idle expiry.
				return nil
			})
			want := 1
			if stalled {
				want = 2
			}
			if err != nil || attempts != want || answer.Len() != 10 {
				t.Fatalf("attempts=%d answer=%s err=%v", attempts, answer.String(), err)
			}
		})
	}
}

func TestIdleRetryAfterWaitIsCancellable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		conn, err := connection.New(connection.VLLM, "http://localhost:8000")
		if err != nil {
			t.Fatal(err)
		}
		client, err := NewClient(conn, "")
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()
		attempts := 0
		client.transport.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": {"30"}}, Body: io.NopCloser(strings.NewReader(`{"error":"busy"}`))}, nil
		})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		timer := time.AfterFunc(time.Second, cancel)
		defer timer.Stop()
		err = client.Generate(ctx, recoveryRequest(), func(modelapi.Event) error { return nil })
		if !errors.Is(err, context.Canceled) || attempts != 1 {
			t.Fatalf("attempts=%d err=%v", attempts, err)
		}
	})
}
