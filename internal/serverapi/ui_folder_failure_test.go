// Purpose: Distinguish unsupported desktops, failed dialogs and deadlines without leaking adapter errors.
// Pattern: HTTP failure-contract regression test.
// Usage: go test ./internal/serverapi -run TestUIFolderFailure.
// Related: internal/serverapi/ui_folder.go, internal/diagnostics/catalog/desktop.go.
package serverapi

import (
	"context"
	"errors"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUIFolderFailureClassification(t *testing.T) {
	f := newFixture(t, nil, false, 1)
	transport := nativeTransport(t, f, "sse")
	if err := transport.enableUI(http.NotFoundHandler()); err != nil {
		t.Fatal(err)
	}
	request := func(path string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequestWithContext(t.Context(), "POST", "http://localhost:8787"+path, nil)
		r.RemoteAddr = "127.0.0.1:3000"
		r.Header.Set("Origin", "http://localhost:8787")
		r.Header.Set("X-Meca-UI", "1")
		if cookie == nil {
			r.Header.Set("Authorization", "Bearer "+testToken)
		} else {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		transport.ServeHTTP(w, r)
		return w
	}
	login := request("/v1/ui/login", nil)
	if login.Code != 204 {
		t.Fatal(login.Code)
	}
	cookie := login.Result().Cookies()[0]
	for _, test := range []struct {
		name   string
		err    error
		status int
	}{
		{"unsupported", fault.New(diag.Unavailable, diag.FolderPickerUnavailable), 503},
		{"failed", errors.New("private desktop diagnostic"), 500},
		{"timeout", context.DeadlineExceeded, 504},
	} {
		t.Run(test.name, func(t *testing.T) {
			f.api.PickFolder = func(context.Context) (string, error) { return "", test.err }
			result := request("/v1/ui/pick-folder", cookie)
			if result.Code != test.status || strings.Contains(result.Body.String(), "private desktop") {
				t.Fatal(result.Code, result.Body.String())
			}
		})
	}
}
