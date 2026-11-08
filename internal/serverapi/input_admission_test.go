// Purpose: Keep tiny controls available while an independent chunked read waits.
// Pattern: Deterministic transport admission regression test.
// Usage: go test ./internal/serverapi -run TestChunkedControlAdmission.
// Related: internal/serverapi/transport.go, internal/serverapi/admission/input.go.
package serverapi

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type eofNoticeBody struct {
	io.Reader
	read chan struct{}
	once sync.Once
}

func (b *eofNoticeBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if err == io.EOF {
		b.once.Do(func() { close(b.read) })
	}
	return n, err
}
func (*eofNoticeBody) Close() error { return nil }

func TestChunkedControlAdmission(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport, err := newTransport(t.Context(), f.api, "sse", testToken, []string{"localhost:8787"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	release, err := f.api.readLimit().Acquire(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	body := &eofNoticeBody{Reader: strings.NewReader(`{"id":"first","listSkills":{}}`), read: make(chan struct{})}
	first := httptest.NewRequestWithContext(ctx, "POST", "http://localhost:8787/v1/command", body)
	first.ContentLength = -1
	first.Header.Set("Authorization", "Bearer "+testToken)
	first.Header.Set("Content-Type", "application/json")
	done := make(chan struct{})
	go func() { defer close(done); transport.ServeHTTP(httptest.NewRecorder(), first) }()
	defer func() { cancel(); <-done }()
	<-body.read
	second := httptest.NewRequestWithContext(ctx, "POST", "http://localhost:8787/v1/command", strings.NewReader(`{"id":"second","state":{}}`))
	second.ContentLength = -1
	second.Header = first.Header.Clone()
	writer := httptest.NewRecorder()
	transport.ServeHTTP(writer, second)
	if writer.Code != 200 {
		t.Fatalf("independent control: %d %s", writer.Code, writer.Body)
	}
}
