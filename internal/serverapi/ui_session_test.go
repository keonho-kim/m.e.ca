// Purpose: Verify anonymous browser session checks do not issue protected commands or weaken authentication.
// Pattern: Browser authentication lifecycle contract tests.
// Usage: go test ./internal/serverapi -run TestUIBrowserSessionStatus.
// Related: internal/serverapi/ui_auth.go, internal/serverapi/security.go.
package serverapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUIBrowserSessionStatus(t *testing.T) {
	f := newFixture(t, nil, false, 1)
	transport := nativeTransport(t, f, "sse")
	if err := transport.enableUI(http.NotFoundHandler()); err != nil {
		t.Fatal(err)
	}
	check := func(cookie *http.Cookie, want bool) {
		t.Helper()
		r := httptest.NewRequestWithContext(t.Context(), "GET", "http://localhost:8787/v1/ui/session", nil)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		transport.ServeHTTP(w, r)
		var result struct {
			Authenticated bool `json:"authenticated"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 || result.Authenticated != want {
			t.Fatalf("browser session status: code=%d authenticated=%v decode=%v", w.Code, result.Authenticated, err)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("browser session status was cacheable")
		}
	}
	check(nil, false)
	check(&http.Cookie{Name: browserCookieName(httptest.NewRequest("GET", "http://localhost:8787/v1/ui/session", nil)), Value: "expired-browser-session-credential"}, false)
	r := httptest.NewRequestWithContext(t.Context(), "POST", "http://localhost:8787/v1/ui/login", nil)
	r.Header.Set("Authorization", "Bearer "+testToken)
	r.Header.Set("Origin", "http://localhost:8787")
	r.Header.Set("X-Meca-UI", "1")
	w := httptest.NewRecorder()
	transport.ServeHTTP(w, r)
	if w.Code != 204 || len(w.Result().Cookies()) != 1 {
		t.Fatal("browser login failed", w.Code)
	}
	check(w.Result().Cookies()[0], true)
	denied := httptest.NewRecorder()
	transport.ServeHTTP(denied, httptest.NewRequestWithContext(t.Context(), "POST", "http://localhost:8787/v1/command", nil))
	if denied.Code != http.StatusUnauthorized {
		t.Fatal("anonymous command bypassed authentication", denied.Code)
	}
}

func TestUIBrowserCookiesSeparateServerPorts(t *testing.T) {
	f := newFixture(t, nil, false, 1)
	transport, err := newTransport(t.Context(), f.api, "sse", testToken, []string{"localhost:8787", "localhost:8788"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(transport.Close)
	if err := transport.enableUI(http.NotFoundHandler()); err != nil {
		t.Fatal(err)
	}
	login := func(authority string) *http.Cookie {
		t.Helper()
		r := httptest.NewRequestWithContext(t.Context(), "POST", "http://"+authority+"/v1/ui/login", nil)
		r.Header.Set("Authorization", "Bearer "+testToken)
		r.Header.Set("Origin", "http://"+authority)
		r.Header.Set("X-Meca-UI", "1")
		w := httptest.NewRecorder()
		transport.ServeHTTP(w, r)
		if w.Code != 204 || len(w.Result().Cookies()) != 1 {
			t.Fatal("browser login failed", w.Code)
		}
		return w.Result().Cookies()[0]
	}
	first, second := login("localhost:8787"), login("localhost:8788")
	if first.Name == second.Name {
		t.Fatal("different server ports overwrite the same browser cookie")
	}
}
