// Purpose: Stop keepalive-only and empty-tool streams without timing out healthy slow consumers.
// Pattern: End-to-end transport lifecycle tests with virtual time.
// Usage: go test -race ./internal/provider/httpclient -run TestSemanticIdle.
// Related: internal/provider/httpclient/request.go, internal/provider/transport/content_watch.go.
package httpclient

import (
	"context"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

type heartbeatBody struct {
	ctx   context.Context
	frame string
}

func (b *heartbeatBody) Read(p []byte) (int, error) {
	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-b.ctx.Done():
		return 0, context.Cause(b.ctx)
	case <-timer.C:
	}
	return copy(p, b.frame), nil
}
func (*heartbeatBody) Close() error { return nil }

func TestSemanticIdleRejectsKeepaliveAndEmptyFunctionFragments(t *testing.T) {
	for _, frame := range []string{": ping\n\n", "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\"}}]}}]}\n\n"} {
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
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: &heartbeatBody{ctx: r.Context(), frame: frame}}, nil
			})
			request := recoveryRequest()
			request.IdleTimeout = time.Second
			started := time.Now()
			err = client.Generate(t.Context(), request, func(modelapi.Event) error { return nil })
			if err == nil || attempts != 3 || time.Since(started) > 5*time.Second {
				t.Fatal("no-content connection remained alive", attempts, err)
			}
		})
	}
}
