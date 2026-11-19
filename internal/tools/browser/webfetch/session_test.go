// Purpose: Verify page budgets, Unicode continuations and temporary artifact ownership.
// Pattern: Single-fetch loopback document with observable cursor and disk state.
// Usage: go test ./internal/tools/browser/webfetch -run TestForwardOnlyDocument.
// Related: internal/tools/browser/webfetch/documents.go, internal/tools/observation/body.go.
package webfetch

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestForwardOnlyDocument(t *testing.T) {
	original := strings.Repeat("가😀나\\\n", 24<<10)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		io.WriteString(w, original)
	}))
	defer server.Close()
	client := NewClient()
	session := NewDocuments(client, t.TempDir(), func(s string) string { return s }, nil)
	input := mustPrepare(t, session, Input{URL: server.URL})
	if _, err := session.Read(toolapi.WithWebFetchDecision(t.Context(), 0, server.URL), input, staticHTML); err == nil || requests.Load() != 0 {
		t.Fatal("a page without context budget initiated a fetch")
	}
	ctx := toolapi.WithWebFetchDecision(t.Context(), 1024, server.URL)
	first, err := session.Read(ctx, input, staticHTML)
	if err != nil || first.NextOffset <= 0 || !utf8.ValidString(first.Text) {
		t.Fatalf("first = %+v, err = %v", first, err)
	}
	encoded, err := json.Marshal(first)
	if err != nil || len(encoded) > 1024 {
		t.Fatalf("tool content exceeded allowance: %d, %v", len(encoded), err)
	}
	var body strings.Builder
	body.WriteString(first.Text)
	next := first.NextOffset
	dead := mustPrepare(t, session, Input{ResultID: first.ResultID, Revision: first.Revision, Offset: &next})
	if _, err := session.Read(toolapi.WithWebFetchDecision(t.Context(), 0, server.URL), dead, staticHTML); err == nil {
		t.Fatal("consumed existing cursor without context budget")
	}
	for next != 0 {
		request := mustPrepare(t, session, Input{ResultID: first.ResultID, Revision: first.Revision, Offset: &next, Limit: new(MaxPageBytes - 1)})
		page, err := session.Read(t.Context(), request, staticHTML)
		if err != nil || !utf8.ValidString(page.Text) || page.Offset != next {
			t.Fatalf("page at %d: %v", next, err)
		}
		next = page.NextOffset
		body.WriteString(page.Text)
	}
	if body.String() != strings.TrimSpace(original) || requests.Load() != 1 {
		t.Fatal("forward pages lost bytes or downloaded the document again")
	}
	if _, err := session.Read(t.Context(), dead, staticHTML); err == nil {
		t.Fatal("replayed a consumed cursor")
	}
	files, err := os.ReadDir(session.work)
	if err != nil || len(files) != 0 {
		t.Fatalf("completed document left %d files: %v", len(files), err)
	}
	first, err = session.Read(ctx, input, staticHTML)
	if err != nil || first.NextOffset <= 0 {
		t.Fatal("fixture did not produce a pending document")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	cursors := session.Cursors()
	if len(cursors) != 0 {
		t.Fatalf("request cleanup kept cursors: %v", cursors)
	}
}

func BenchmarkDocumentFirstPage(b *testing.B) {
	const paragraph = "<p>Read the <a href='/guide'>local guide</a> and note the status.</p>"
	for _, repeats := range []int{8, 4096} {
		b.Run(strconv.Itoa(repeats)+"_paragraphs", func(b *testing.B) {
			source := "<title>Manual</title><main>" + strings.Repeat(paragraph, repeats) + "</main>"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				io.WriteString(w, source)
			}))
			defer server.Close()
			client := NewClient()
			session := NewDocuments(client, b.TempDir(), func(s string) string { return s }, nil)
			request, err := session.Prepare(Input{URL: server.URL})
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for b.Loop() {
				if _, err := session.Read(userURLContext(b.Context(), server.URL), request, staticHTML); err != nil {
					b.Fatal(err)
				}
				if err := session.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})

	}
}
