// Purpose: Keep authenticated observation lifetimes bound to browser credential revocation.
// Pattern: HTTP integration regression with an owned streaming handler.
// Usage: go test ./internal/serverapi -run TestUIRevocation.
// Related: internal/serverapi/ui_auth.go, internal/serverapi/transport.go.
package serverapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUIRevocationCancelsAdmittedObservation(t *testing.T) {
	f := newFixture(t, nil, false, 1)
	transport := nativeTransport(t, f, "sse")
	if err := transport.enableUI(http.NotFoundHandler()); err != nil {
		t.Fatal(err)
	}
	request := func(path string, cookie *http.Cookie) *http.Request {
		r := httptest.NewRequestWithContext(t.Context(), "POST", "http://localhost:8787"+path, nil)
		r.Header.Set("Origin", "http://localhost:8787")
		r.Header.Set("X-Meca-UI", "1")
		if cookie == nil {
			r.Header.Set("Authorization", "Bearer "+testToken)
		} else {
			r.AddCookie(cookie)
		}
		return r
	}
	login := httptest.NewRecorder()
	transport.ServeHTTP(login, request("/v1/ui/login", nil))
	if login.Code != 204 {
		t.Fatal(login.Code)
	}
	cookie := login.Result().Cookies()[0]
	entered, finished := make(chan struct{}), make(chan struct{})
	transport.handler.(*http.ServeMux).HandleFunc("POST /v1/observe-test", func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done(); close(finished) })
	go transport.ServeHTTP(httptest.NewRecorder(), request("/v1/observe-test", cookie))
	<-entered
	logout := httptest.NewRecorder()
	transport.ServeHTTP(logout, request("/v1/ui/logout", cookie))
	if logout.Code != 204 {
		t.Fatal(logout.Code)
	}
	select {
	case <-finished:
	case <-deadline(t).Done():
		t.Fatal("revoked browser request is still observing")
	}
}
