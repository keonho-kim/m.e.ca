// Purpose: Keep a large-result slot until its consumer finishes writing and reject excess input before admission.
// Pattern: Channel-controlled HTTP resource-lifetime tests.
// Usage: go test -race ./internal/serverapi -run 'TestResultSlot|TestInputBudget'.
// Related: internal/serverapi/openai/result.go, internal/serverapi/transport.go.
package serverapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
	"github.com/keonho-kim/meca/internal/serverapi/openai"
)

type heldResultWriter struct {
	header           http.Header
	entered, release chan struct{}
	once             sync.Once
}

func (w *heldResultWriter) Header() http.Header { return w.header }
func (*heldResultWriter) WriteHeader(int)       {}
func (w *heldResultWriter) Write(body []byte) (int, error) {
	w.once.Do(func() { close(w.entered) })
	<-w.release
	return len(body), nil
}

func TestResultSlotCoversNetworkWrite(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	accepted, err := f.app.Submit(t.Context(), server.Submission{Kind: "response", Stored: true, Messages: []server.InputMessage{{Role: "user", Text: "answer"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.WaitResponse(t.Context(), accepted.Response.ID); err != nil {
		t.Fatal(err)
	}
	h := &openai.Handler{Service: f.app, Redact: f.api.Redact, Reads: admission.New(1)}
	mux := http.NewServeMux()
	h.Mount(mux)
	w := &heldResultWriter{header: make(http.Header), entered: make(chan struct{}), release: make(chan struct{})}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(w.release) }) }
	defer unblock()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		mux.ServeHTTP(w, httptest.NewRequestWithContext(ctx, "GET", "/v1/responses/"+accepted.Response.ID, nil))
	}()
	select {
	case <-w.entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if extra, ok := h.Reads.Try(1); ok {
		extra()
		t.Error("original-read admission ended before its output was released")
	}
	unblock()
	<-finished
	extra, ok := h.Reads.Try(1)
	if !ok {
		t.Fatal("completed result retained admission")
	}
	extra()
}

func TestInputBudgetRejectsBeforeTaskAdmission(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := openAITransport(t, f, "sse")
	release, ok := transport.inputs.Try(32 << 20)
	if !ok {
		t.Fatal("cannot reserve input budget")
	}
	defer release()
	r := httptest.NewRequest("POST", "http://localhost:8787/v1/responses", strings.NewReader(`{"model":"meca","input":"answer"}`))
	r.Header.Set("Authorization", "Bearer "+testToken)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	transport.ServeHTTP(w, r)
	if w.Code != 429 || f.app.State().Session.ID != "" {
		t.Fatal("input exceeded admission before execution", w.Code, w.Body.String())
	}
}
