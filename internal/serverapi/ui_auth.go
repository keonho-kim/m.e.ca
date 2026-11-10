// Purpose: Issue bounded, expiring browser credentials without exposing the server bearer token.
// Pattern: Process-owned HTTP session adapter.
// Usage: EnableUI mounts explicit login/logout and cookie authentication.
// Related: internal/serverapi/security.go, internal/serverapi/transport.go.
package serverapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

const uiCookie = "meca_ui"
const uiLifetime = 8 * time.Hour

func browserCookieName(r *http.Request) string {
	identity := sha256.Sum256([]byte(strings.ToLower(r.Host)))
	return uiCookie + "_" + hex.EncodeToString(identity[:6])
}

type browserSessions struct {
	mu     sync.Mutex
	values map[[32]byte]time.Time
}

func (t *Transport) enableUI(handler http.Handler) error {
	seed := rand.Text()
	if t.api.Remember != nil {
		if err := t.api.Remember(seed); err != nil {
			return err
		}
	}
	t.uiSeed = seed
	t.ui = handler
	t.browser = &browserSessions{values: make(map[[32]byte]time.Time)}
	mux := t.handler.(*http.ServeMux)
	mux.Handle("GET /ui/", handler)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/ui/", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("POST /v1/ui/login", t.uiLogin)
	mux.HandleFunc("POST /v1/ui/logout", t.uiLogout)
	mux.HandleFunc("GET /v1/ui/session", t.uiSession)
	return nil
}

func (t *Transport) uiSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Authenticated bool `json:"authenticated"`
	}{Authenticated: t.browserAuthenticated(r)})
}

func (t *Transport) uiLogin(w http.ResponseWriter, r *http.Request) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || !t.authenticates(token) {
		http.Error(w, diag.Text(diag.UIAuthRejected), http.StatusUnauthorized)
		return
	}
	now := time.Now()
	// One process-lived masking prefix avoids expanding secret history and
	// invalidating document revisions on every login; each session is still random.
	value := t.uiSeed + "." + rand.Text()
	t.browser.mu.Lock()
	for key, expiry := range t.browser.values {
		if !expiry.After(now) {
			delete(t.browser.values, key)
		}
	}
	if old, err := r.Cookie(browserCookieName(r)); err == nil {
		delete(t.browser.values, sha256.Sum256([]byte(old.Value)))
	}
	if len(t.browser.values) >= 64 {
		t.browser.mu.Unlock()
		http.Error(w, diag.Text(diag.UIAuthCapacity), http.StatusTooManyRequests)
		return
	}
	t.browser.values[sha256.Sum256([]byte(value))] = now.Add(uiLifetime)
	t.browser.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: browserCookieName(r), Value: value, Path: "/v1", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: int(uiLifetime.Seconds())})
	w.WriteHeader(http.StatusNoContent)
}

func (t *Transport) browserAuthenticated(r *http.Request) bool {
	if t.browser == nil {
		return false
	}
	cookie, err := r.Cookie(browserCookieName(r))
	if err != nil || len(cookie.Value) < 26 || len(cookie.Value) > 128 {
		return false
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if r.Header.Get("Origin") != scheme+"://"+r.Host || r.Header.Get("X-Meca-UI") != "1" {
			return false
		}
	}
	key := sha256.Sum256([]byte(cookie.Value))
	t.browser.mu.Lock()
	defer t.browser.mu.Unlock()
	expiry, ok := t.browser.values[key]
	if !ok {
		return false
	}
	if !expiry.After(time.Now()) {
		delete(t.browser.values, key)
		return false
	}
	return true
}

func (t *Transport) uiLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(browserCookieName(r)); err == nil {
		t.browser.mu.Lock()
		delete(t.browser.values, sha256.Sum256([]byte(cookie.Value)))
		t.browser.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: browserCookieName(r), Path: "/v1", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}
