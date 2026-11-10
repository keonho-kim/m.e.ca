// Purpose: Verify public asset routing and cookie/CSRF boundaries preserve API authentication.
// Pattern: HTTP integration contract test.
// Usage: go test ./internal/serverapi -run TestUIAuthentication.
// Related: internal/serverapi/ui_auth.go, internal/serverapi/security.go.
package serverapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUIAuthenticationAndStaticRouting(t *testing.T) {
	f := newFixture(t, nil, false, 1)
	transport := nativeTransport(t, f, "sse")
	transport.enableUI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=123")
		w.WriteHeader(http.StatusOK)
	}))
	request := func(method, path, token string, cookie *http.Cookie, origin, header string) *httptest.ResponseRecorder {
		r := httptest.NewRequestWithContext(t.Context(), method, "http://localhost:8787"+path, strings.NewReader(`{"capabilities":{}}`))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if header != "" {
			r.Header.Set("X-Meca-UI", header)
		}
		w := httptest.NewRecorder()
		transport.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/ui/", "", nil, "", ""); w.Code != 200 || w.Header().Get("Cache-Control") == "no-store" {
		t.Fatal("public immutable assets", w.Code, w.Header())
	}
	if w := request("GET", "/", "", nil, "", ""); w.Code != http.StatusTemporaryRedirect {
		t.Fatal("root redirect", w.Code)
	}
	if w := request("POST", "/v1/command", "", nil, "", ""); w.Code != 401 {
		t.Fatal("anonymous API", w.Code)
	}
	w := request("POST", "/v1/ui/login", testToken, nil, "http://localhost:8787", "1")
	if w.Code != 204 || len(w.Result().Cookies()) != 1 {
		t.Fatal("login", w.Code)
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("cookie flags", cookie)
	}
	if w := request("POST", "/v1/command", "", cookie, "http://localhost:8787", "1"); w.Code != 200 {
		t.Fatal("cookie API", w.Code, w.Body.String())
	}
	if w := request("POST", "/v1/command", "", cookie, "http://localhost:8787", ""); w.Code != 401 {
		t.Fatal("missing CSRF header", w.Code)
	}
	if w := request("POST", "/v1/command", "", cookie, "http://evil.test", "1"); w.Code != 403 {
		t.Fatal("foreign origin", w.Code)
	}
	if w := request("POST", "/v1/ui/logout", "", cookie, "http://localhost:8787", "1"); w.Code != 204 {
		t.Fatal("logout", w.Code)
	}
	if w := request("POST", "/v1/command", "", cookie, "http://localhost:8787", "1"); w.Code != 401 {
		t.Fatal("revoked cookie", w.Code)
	}
}
