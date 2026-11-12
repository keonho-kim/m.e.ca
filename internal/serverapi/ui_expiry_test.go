// Purpose: Revoke already-admitted browser observations when their credential expires.
// Pattern: Deterministic session-clock regression without real waits or sockets.
// Usage: go test ./internal/serverapi -run TestUIExpiry.
// Related: internal/serverapi/browser_sessions.go.
package serverapi

import (
	"context"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestUIExpiryCancelsAdmittedObservation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		value := strings.Repeat("x", 32)
		r := httptest.NewRequestWithContext(ctx, "GET", "http://localhost/v1/events", nil)
		r.AddCookie(&http.Cookie{Name: browserCookieName(r), Value: value})
		session := newBrowserSession(ctx, time.Now().Add(time.Minute))
		transport := &Transport{browser: &browserSessions{values: map[[32]byte]*browserSession{sha256.Sum256([]byte(value)): session}}}
		bound, release, ok := transport.browserRequest(r)
		if !ok || bound.Context().Err() != nil {
			t.Fatal("valid session rejected")
		}
		defer release()
		time.Sleep(time.Minute)
		synctest.Wait()
		if bound.Context().Err() == nil || transport.browserAuthenticated(r) {
			t.Fatal("expired credential retained an observer")
		}
		if len(transport.browser.values) != 0 {
			t.Fatal("expired lookup retained credential")
		}
	})
}
