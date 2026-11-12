// Purpose: Bind browser credentials and their admitted HTTP observers to one expiring lifetime.
// Pattern: Process-owned authentication sessions with request-scoped cancellation.
// Usage: Login creates a session; logout, replacement, expiry and transport shutdown cancel observers.
// Related: internal/serverapi/ui_auth.go, internal/serverapi/security.go, internal/serverapi/transport.go.
package serverapi

import (
	"context"
	"crypto/sha256"
	"net/http"
	"strings"
	"sync"
	"time"
)

type browserSession struct {
	expiry time.Time
	ctx    context.Context
	cancel context.CancelFunc
}
type browserSessions struct {
	mu     sync.Mutex
	values map[[32]byte]*browserSession
}

func newBrowserSession(parent context.Context, expiry time.Time) *browserSession {
	ctx, cancel := context.WithDeadline(parent, expiry)
	return &browserSession{expiry: expiry, ctx: ctx, cancel: cancel}
}

// revokeLocked cancels admitted observers before removing their credential.
func (s *browserSessions) revokeLocked(key [32]byte) {
	if session := s.values[key]; session != nil {
		session.cancel()
		delete(s.values, key)
	}
}
func (t *Transport) browserSession(r *http.Request) *browserSession {
	if t.browser == nil {
		return nil
	}
	cookie, err := r.Cookie(browserCookieName(r))
	if err != nil || len(cookie.Value) < 26 || len(cookie.Value) > 128 {
		return nil
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if r.Header.Get("Origin") != scheme+"://"+r.Host || r.Header.Get("X-Meca-UI") != "1" {
			return nil
		}
	}
	key := sha256.Sum256([]byte(cookie.Value))
	t.browser.mu.Lock()
	defer t.browser.mu.Unlock()
	session := t.browser.values[key]
	if session != nil && (session.ctx.Err() != nil || !session.expiry.After(time.Now())) {
		t.browser.revokeLocked(key)
		return nil
	}
	return session
}
func (t *Transport) browserAuthenticated(r *http.Request) bool { return t.browserSession(r) != nil }

func (t *Transport) browserRequest(r *http.Request) (*http.Request, func(), bool) {
	path := r.URL.Path
	public := r.Method == "OPTIONS" || path == "/" || path == "/ui" || strings.HasPrefix(path, "/ui/") || path == "/v1/ui/session" || path == "/v1/ui/login" || path == "/v1/ui/logout"
	if public || t.socketHandshake(r) || t.bearerAuthenticated(r) {
		return r, func() {}, true
	}
	session := t.browserSession(r)
	if session == nil {
		return r, func() {}, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	stop := context.AfterFunc(session.ctx, cancel)
	return r.WithContext(ctx), func() { stop(); cancel() }, true
}
