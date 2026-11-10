// Purpose: Keep input deadlines from expiring established streaming responses.
// Pattern: HTTP boundary regression tests with recorded read-deadline operations.
// Usage: go test ./internal/serverapi -run TestStreamingInputDeadline.
// Related: internal/serverapi/transport.go, internal/serverapi/http.go.
package serverapi

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type deadlineRecorder struct {
	*httptest.ResponseRecorder
	read      time.Time
	set       bool
	cancel    context.CancelFunc
	flushRead time.Time
}

func (w *deadlineRecorder) SetReadDeadline(value time.Time) error {
	w.read = value
	w.set = w.set || !value.IsZero()
	return nil
}
func (w *deadlineRecorder) FlushError() error {
	w.flushRead = w.read
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

func TestStreamingInputDeadline(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := nativeTransport(t, f, "sse")
	post := httptest.NewRequest("POST", "http://localhost:8787/v1/command", strings.NewReader(`{"state":{}}`))
	post.Header.Set("Authorization", "Bearer "+testToken)
	post.Header.Set("Content-Type", "application/json")
	writer := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	transport.ServeHTTP(writer, post)
	if !writer.set || !writer.read.IsZero() || !writer.flushRead.IsZero() || writer.Code != 200 {
		t.Fatal(writer.set, writer.read, writer.Code)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	get := httptest.NewRequestWithContext(ctx, "GET", "http://localhost:8787/v1/events", nil)
	get.Header.Set("Authorization", "Bearer "+testToken)
	writer = &deadlineRecorder{ResponseRecorder: httptest.NewRecorder(), cancel: cancel}
	transport.ServeHTTP(writer, get)
	if writer.set {
		t.Fatal("idle GET stream received an input deadline")
	}
}
