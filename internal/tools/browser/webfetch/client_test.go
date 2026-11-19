// Purpose: Verify private transport and bounded document handling against loopback fixtures.
// Pattern: HTTP adapter contract tests.
// Usage: go test ./internal/tools/browser/webfetch.
// Related: internal/tools/browser/webfetch/client.go, internal/tools/browser/webfetch/document.go.
package webfetch

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/browser/chromium"
)

func TestPinnedNameBypassesAmbientProxy(t *testing.T) {
	var direct, proxyHits atomic.Int32
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		direct.Add(1)
		if !strings.HasPrefix(request.Host, "manual.invalid:") {
			t.Errorf("HTTP host = %q", request.Host)
		}
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "private manual")
	}))
	defer origin.Close()
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		proxyHits.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer proxy.Close()
	t.Setenv("HTTP_PROXY", proxy.URL)
	t.Setenv("HTTPS_PROXY", proxy.URL)
	t.Setenv("NO_PROXY", "")
	host, port, err := net.SplitHostPort(origin.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	address := "http://manual.invalid:" + port
	client := NewClient()
	client.lookup = func(_ context.Context, _, name string) ([]netip.Addr, error) {
		if name != "manual.invalid" {
			t.Fatalf("unexpected DNS lookup %q", name)
		}
		return []netip.Addr{netip.MustParseAddr(host)}, nil
	}
	session := NewDocuments(client, t.TempDir(), func(s string) string { return s }, nil)
	page, err := session.Read(userURLContext(t.Context(), address), mustPrepare(t, session, Input{URL: address}), staticHTML)
	if err != nil || page.Text != "private manual" || direct.Load() != 1 || proxyHits.Load() != 0 {
		t.Fatalf("direct=%d proxy=%d result=%+v error=%v", direct.Load(), proxyHits.Load(), page, err)
	}
}

func TestPrivateFetch(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/article?issued=private-query", http.StatusFound)
		case "/escape":
			http.Redirect(w, r, "http://public.example.invalid/article", http.StatusFound)
		case "/article":
			if r.Referer() != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
				t.Error("ambient request state reached the document server")
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, `<title>내부 문서</title><nav>menu-only</nav><main><h1>Guide</h1><p>Read <a href="/docs">the manual</a> and the <a href="#example">example</a>.</p><table><tr><th>Name</th><th>Count</th></tr><tr><td>Files</td><td>7</td></tr></table><ul><li>First item</li></ul><pre><code class="language-go">fmt.Println("hello")</code></pre><script>untrusted script</script><img src="http://public.example.invalid/pixel"></main>`)
		}
	}))
	defer server.Close()
	client := NewClient()
	session := NewDocuments(client, t.TempDir(), func(s string) string { return s }, nil)
	address := server.URL + "/redirect?secret=hidden"
	request := mustPrepare(t, session, Input{URL: address})
	for _, ctx := range []context.Context{
		t.Context(),
		userURLContext(t.Context(), server.URL+"/redirect"),
		userURLContext(t.Context(), "https://public.example/?next="+address),
	} {
		if _, err := session.Read(ctx, request, staticHTML); err == nil || calls.Load() != 0 {
			t.Fatal("an undisclosed private URL reached the target")
		}
	}
	ctx := toolapi.WithWebFetchDecision(t.Context(), 64<<10, "[문서]("+address+") 읽어줘")
	page, err := session.Read(ctx, request, staticHTML)
	if err != nil {
		t.Fatal(err)
	}
	if !page.Untrusted || page.Title != "내부 문서" || page.URL != server.URL+"/article" || page.NextOffset != 0 {
		t.Fatalf("first page = %+v", page)
	}
	for _, want := range []string{"# Guide", "[the manual](" + server.URL + "/docs)", "| Name | Count |", "| Files | 7 |", "- First item", "```go\nfmt.Println"} {
		if !strings.Contains(page.Text, want) {
			t.Errorf("missing %q in %s", want, page.Text)
		}
	}
	if strings.Contains(page.Text, "menu-only") || strings.Contains(page.Text, "untrusted script") || strings.Contains(page.Text, "private-query") {
		t.Fatal("non-content or a redirect query appeared in the extracted document")
	}
	if _, err := session.Read(userURLContext(t.Context(), server.URL+"/escape"), mustPrepare(t, session, Input{URL: server.URL + "/escape"}), staticHTML); err == nil {
		t.Fatal("permitted a redirect outside the allowed origins")
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("HTTP requests = %d; want only explicit loopback requests", got)
	}
}

func TestPublicFetchAndSameOriginResources(t *testing.T) {
	var bodyHits, resourceHits, privateHits, connections atomic.Int32
	const firstHost, finalHost = "public-docs.invalid", "manual-mirror.invalid"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Referer() != "" {
			t.Error("request forwarded extra state")
		}
		_, port, err := net.SplitHostPort(r.Host)
		if err != nil {
			t.Errorf("invalid fixture authority: %v", err)
			http.Error(w, "bad authority", http.StatusBadRequest)
			return
		}
		switch r.URL.Path {
		case "/start":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "server-supplied"})
			http.Redirect(w, r, "http://"+finalHost+":"+port+"/manual", http.StatusFound)
		case "/manual":
			if r.Host != finalHost+":"+port {
				t.Error("validated IP replaced the HTTP authority")
			}
			bodyHits.Add(1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, "<main><p>public document</p></main>")
		case "/needed.js":
			resourceHits.Add(1)
			w.Header().Set("Content-Type", "application/javascript")
			io.WriteString(w, "document.title = 'manual'")
		case "/private-redirect":
			http.Redirect(w, r, "http://127.0.0.1:"+port+"/internal", http.StatusFound)
		case "/internal":
			privateHits.Add(1)
		default:
			t.Errorf("unexpected request for %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	ipByHost := map[string]netip.Addr{
		firstHost: netip.MustParseAddr("93.184.215.11"),
		finalHost: netip.MustParseAddr("93.184.215.12"),
	}
	client := NewClient()
	lookups := 0
	client.lookup = func(_ context.Context, _, host string) ([]netip.Addr, error) {
		lookups++
		if ip, ok := ipByHost[host]; ok {
			return []netip.Addr{ip}, nil
		}
		t.Errorf("unexpected DNS query for %q", host)
		return nil, fmt.Errorf("unexpected DNS lookup for %s", host)
	}
	client.dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		connections.Add(1)
		if address != net.JoinHostPort(ipByHost[firstHost].String(), port) && address != net.JoinHostPort(ipByHost[finalHost].String(), port) {
			t.Errorf("dial target escaped validated public addresses: %s", address)
			return nil, fmt.Errorf("unexpected target %s", address)
		}
		// A loopback fixture stands in for the public listener. Production dials
		// the literal address given to this boundary.
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	documents := NewDocuments(client, t.TempDir(), func(s string) string { return s }, nil)
	start := "http://" + firstHost + ":" + port
	end := "http://" + finalHost + ":" + port
	render := func(ctx context.Context, address string, main chromium.Response, load chromium.LoadResource) (string, error) {
		if address != end+"/manual" {
			t.Errorf("renderer used unexpected final URL %q", address)
		}
		if _, err := load(ctx, end+"/needed.js", http.MethodGet, 4096); err != nil {
			return "", err
		}
		if _, err := load(ctx, "http://unrelated.invalid/collect", http.MethodGet, 4096); err == nil {
			t.Error("a cross-origin render request was accepted")
		}
		return string(main.Body), nil
	}
	page, err := documents.Read(t.Context(), mustPrepare(t, documents, Input{URL: start + "/start"}), render)
	if err != nil || page.URL != end+"/manual" || page.Text != "public document" ||
		lookups != 2 || connections.Load() != 2 || bodyHits.Load() != 1 || resourceHits.Load() != 1 {
		t.Fatalf("public read failed: page=%+v lookups=%d connections=%d bodies=%d resources=%d err=%v", page, lookups, connections.Load(), bodyHits.Load(), resourceHits.Load(), err)
	}
	if _, err := documents.Read(t.Context(), mustPrepare(t, documents, Input{URL: start + "/private-redirect"}), render); err == nil || privateHits.Load() != 0 {
		t.Fatal("a public redirect entered a private network")
	}
}

func TestHTTPSUsesSystemTrustWithoutCAConfiguration(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	client := NewClient()
	documents := NewDocuments(client, t.TempDir(), func(s string) string { return s }, nil)
	if _, err := documents.Read(userURLContext(t.Context(), server.URL), mustPrepare(t, documents, Input{URL: server.URL}), nil); err == nil || hits.Load() != 0 {
		t.Fatalf("untrusted server reached HTTP handler: calls=%d error=%v", hits.Load(), err)
	}
}

func TestDecodedBodyLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "gzip")
		encoded := gzip.NewWriter(w)
		io.WriteString(encoded, strings.Repeat("a", MaxDocumentBytes+1))
		encoded.Close()
	}))
	defer server.Close()
	client := NewClient()
	session := NewDocuments(client, t.TempDir(), func(s string) string { return s }, nil)
	if _, err := session.Read(userURLContext(t.Context(), server.URL), mustPrepare(t, session, Input{URL: server.URL}), staticHTML); err == nil {
		t.Fatal("accepted an oversized decompressed body")
	}
	if cursors := session.Cursors(); len(cursors) != 0 {
		t.Fatalf("failed document retained a cursor: %v", cursors)
	}
}

func mustPrepare(t *testing.T, session *Documents, input Input) Request {
	t.Helper()
	request, err := session.Prepare(input)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func staticHTML(_ context.Context, _ string, main chromium.Response, _ chromium.LoadResource) (string, error) {
	return string(main.Body), nil
}

func userURLContext(ctx context.Context, target string) context.Context {
	return toolapi.WithWebFetchDecision(ctx, 64<<10, "Read "+target)
}
