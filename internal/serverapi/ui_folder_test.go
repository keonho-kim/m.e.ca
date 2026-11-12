// Purpose: Keep native desktop selection behind browser authentication and local address checks.
// Pattern: HTTP boundary regression tests with a controllable desktop adapter.
// Usage: go test ./internal/serverapi -run TestUIFolder.
// Related: internal/serverapi/ui_folder.go, internal/serverapi/ui_auth.go.
package serverapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestUIFolderLocalBrowserOnly(t *testing.T) {
	f := newFixture(t, nil, false, 1)
	transport := nativeTransport(t, f, "sse")
	if err := transport.enableUI(http.NotFoundHandler()); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	f.api.PickFolder = func(context.Context) (string, error) { calls.Add(1); return "/tmp/chosen", nil }
	request := func(remote string, cookie *http.Cookie, csrf bool) *httptest.ResponseRecorder {
		r := httptest.NewRequestWithContext(t.Context(), "POST", "http://localhost:8787/v1/ui/pick-folder", nil)
		r.RemoteAddr = remote
		r.Header.Set("Origin", "http://localhost:8787")
		if csrf {
			r.Header.Set("X-Meca-UI", "1")
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		transport.ServeHTTP(w, r)
		return w
	}
	r := httptest.NewRequestWithContext(t.Context(), "POST", "http://localhost:8787/v1/ui/login", nil)
	r.Header.Set("Authorization", "Bearer "+testToken)
	r.Header.Set("Origin", "http://localhost:8787")
	r.Header.Set("X-Meca-UI", "1")
	w := httptest.NewRecorder()
	transport.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	cookie := w.Result().Cookies()[0]
	for _, tc := range []struct {
		remote string
		cookie *http.Cookie
		csrf   bool
		status int
	}{
		{"127.0.0.1:3000", nil, true, 401},
		{"127.0.0.1:3000", cookie, false, 401},
		{"192.0.2.2:3000", cookie, true, 403},
		{"[::1]:3000", cookie, true, 200},
	} {
		if reply := request(tc.remote, tc.cookie, tc.csrf); reply.Code != tc.status {
			t.Fatal(tc.remote, reply.Code, reply.Body.String())
		}
	}
	if calls.Load() != 1 {
		t.Fatal("rejected request opened a desktop dialog", calls.Load())
	}
	started, release := make(chan struct{}), make(chan struct{})
	f.api.PickFolder = func(ctx context.Context) (string, error) {
		close(started)
		select {
		case <-release:
			return "", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- request("127.0.0.1:3000", cookie, true) }()
	select {
	case <-started:
	case result := <-done:
		t.Fatal("picker returned before opening", result.Code)
	}
	if reply := request("127.0.0.1:3000", cookie, true); reply.Code != 409 {
		t.Fatal(reply.Code)
	}
	close(release)
	if result := <-done; result.Code != 200 {
		t.Fatal(result.Code)
	}
	f.api.PickFolder = func(context.Context) (string, error) { return "", nil }
	if reply := request("127.0.0.1:3000", cookie, true); reply.Code != 200 || reply.Body.String() != "{\"path\":\"\"}\n" {
		t.Fatal(reply.Code, reply.Body.String())
	}
	f.api.PickFolder = nil
	if reply := request("127.0.0.1:3000", cookie, true); reply.Code != 503 {
		t.Fatal(reply.Code)
	}
}

func TestUIFolderLocalAuthorityWithoutExplicitPort(t *testing.T) {
	f := newFixture(t, nil, false, 1)
	transport := nativeTransport(t, f, "sse")
	transport.hosts["localhost"] = true
	if err := transport.enableUI(http.NotFoundHandler()); err != nil {
		t.Fatal(err)
	}
	f.api.PickFolder = func(context.Context) (string, error) { return "", nil }
	request := func(path string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "http://localhost"+path, nil)
		r.RemoteAddr = "127.0.0.1:3000"
		r.Header.Set("Origin", "http://localhost")
		r.Header.Set("X-Meca-UI", "1")
		if cookie != nil {
			r.AddCookie(cookie)
		} else {
			r.Header.Set("Authorization", "Bearer "+testToken)
		}
		w := httptest.NewRecorder()
		transport.ServeHTTP(w, r)
		return w
	}
	login := request("/v1/ui/login", nil)
	if login.Code != 204 {
		t.Fatal(login.Code)
	}
	if result := request("/v1/ui/pick-folder", login.Result().Cookies()[0]); result.Code != 200 {
		t.Fatal("local default-port authority rejected", result.Code)
	}
}
