// Purpose: Exercise Chromium rendering, exclusive network admission and forward-only pages.
// Pattern: Native integration with loopback fixtures on both sides of the allowlist.
// Usage: MECA_TEST_NATIVE_WEBFETCH=1 go test ./internal/tools/browser -run TestNativeWebFetch.
// Related: internal/tools/browser/webfetch.go, internal/tools/browser/chromium/resources.go.
package browser

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/browser/runtime"
	"github.com/keonho-kim/meca/internal/tools/browser/webfetch"
)

func TestNativeWebFetch(t *testing.T) {
	if os.Getenv("MECA_TEST_NATIVE_WEBFETCH") != "1" {
		t.Skip("set MECA_TEST_NATIVE_WEBFETCH=1 to use installed Chromium")
	}
	state, artifacts := t.TempDir(), t.TempDir()
	if !runtime.ChromeAvailable(state) {
		t.Fatal("the native renderer test needs installed Chromium")
	}
	var mainHits, scriptHits, dataHits, deniedHits, writeHits atomic.Int32
	denied := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deniedHits.Add(1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer denied.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.UserAgent() != "MECA WebFetch" || r.Header.Get("Cookie") != "" || r.Header.Get("Referer") != "" {
			t.Error("a request bypassed the private Go transport or forwarded browser state")
		}
		if r.Method != http.MethodGet || r.URL.Path == "/mutate" {
			writeHits.Add(1)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case "/manual":
			mainHits.Add(1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, `<title>Before rendering</title><main><p id="result">Loading</p></main><script src="/app.js"></script>`)
		case "/app.js":
			scriptHits.Add(1)
			w.Header().Set("Content-Type", "application/javascript")
			fmt.Fprintf(w, `
fetch(%q).catch(() => {});
fetch("/redirect-to-denied").catch(() => {});
fetch("/mutate", {method:"POST", body:"test"}).catch(() => {});
new WebSocket(%q).onerror = () => {};
fetch("/data").then(r => r.json()).then(data => {
  document.title = "Rendered title";
  document.getElementById("result").textContent = data.text;
  const scriptResult = document.createElement("p");
  scriptResult.textContent = (typeof RTCPeerConnection === "undefined" ? "RTC blocked " : "RTC allowed ") + "HTML detail ".repeat(2000);
  document.querySelector("main").append(scriptResult);
});
`, denied.URL+"/external", "ws"+strings.TrimPrefix(denied.URL, "http")+"/socket")
		case "/data":
			dataHits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"text":"본문이 JavaScript에서 생성됐습니다."}`)
		case "/redirect-to-denied":
			http.Redirect(w, r, denied.URL, http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := webfetch.NewClient()
	session := New(state, artifacts, func(value string) string { return value }, nil, client)
	defer session.Close(t.Context())
	if _, documents := session.AvailableTools(); !documents {
		t.Fatal("installed Chromium with approved origins was not advertised")
	}
	prepared, err := session.PrepareDocument(webfetch.Input{URL: server.URL + "/manual"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := toolapi.WithWebFetchDecision(t.Context(), 64<<10, server.URL+"/manual")
	first, err := session.ReadDocument(ctx, prepared)
	if err != nil {
		t.Fatal(err)
	}
	if first.Title != "Rendered title" || !strings.Contains(first.Text, "생성됐습니다") || !strings.Contains(first.Text, "RTC blocked") || first.NextOffset <= 0 {
		t.Fatalf("unexpected rendered document title=%q first_bytes=%d next=%d", first.Title, len(first.Text), first.NextOffset)
	}
	live, err := session.Cursors(t.Context())
	if err != nil || len(live) != 1 || live[0].Tool != toolapi.WebFetch {
		t.Fatalf("browser session did not expose the document cursor: %+v %v", live, err)
	}
	next, err := session.PrepareDocument(webfetch.Input{ResultID: first.ResultID, Revision: first.Revision, Offset: &first.NextOffset})
	if err != nil {
		t.Fatal(err)
	}
	page, err := session.ReadDocument(t.Context(), next)
	if err != nil || !strings.Contains(page.Text, "HTML detail") {
		t.Fatalf("continuation did not read the stored render: %v", err)
	}
	if mainHits.Load() != 1 || scriptHits.Load() != 1 || dataHits.Load() != 1 || deniedHits.Load() != 0 || writeHits.Load() != 0 {
		t.Fatalf("unexpected requests main=%d script=%d data=%d denied=%d write=%d", mainHits.Load(), scriptHits.Load(), dataHits.Load(), deniedHits.Load(), writeHits.Load())
	}
	entries, err := os.ReadDir(artifacts)
	if err != nil || len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "web-observations-") {
		t.Fatalf("browser resources survived rendering: %v %v", entries, err)
	}
	if err := session.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(artifacts); err != nil || len(entries) != 0 {
		t.Fatalf("browser document cleanup left artifacts: %v %v", entries, err)
	}
}
