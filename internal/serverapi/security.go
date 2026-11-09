// Purpose: Authenticate requests and enforce exact Host, Origin and CORS trust boundaries.
// Pattern: HTTP security adapter.
// Usage: Transport.ServeHTTP calls authorize before dispatch or resource admission.
// Related: internal/serverapi/transport.go, internal/serverapi/socket.go.
package serverapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func validHost(host string) bool {
	if strings.ContainsAny(host, "* /\\@?#\r\n\t") {
		return false
	}
	u, err := url.Parse("http://" + host)
	if err != nil || u.Host != host || u.Hostname() == "" || u.User != nil {
		return false
	}
	if strings.Contains(u.Hostname(), ":") && net.ParseIP(u.Hostname()) == nil {
		return false
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		return err == nil && n > 0 && n <= 65535
	}
	return !strings.HasSuffix(host, ":")
}
func validOrigin(origin string) bool {
	u, err := url.Parse(origin)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && validHost(u.Host) && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
}
func (t *Transport) authenticates(token string) bool {
	got := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(got[:], t.token[:]) == 1
}

func (t *Transport) authorize(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !t.hosts[strings.ToLower(r.Host)] {
		http.Error(w, diagcatalog.Text(diagcatalog.ServerapiHostIsNotAllowed), http.StatusForbidden)
		return false
	}
	if r.URL.RawQuery != "" && r.URL.Path != "/v1/events" {
		http.Error(w, diagcatalog.Text(diagcatalog.ServerapiQueryParametersAreNotSupported), http.StatusBadRequest)
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if !t.origins[origin] && origin != scheme+"://"+r.Host {
			http.Error(w, diagcatalog.Text(diagcatalog.ServerapiOriginIsNotAllowed), http.StatusForbidden)
			return false
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Expose-Headers", "Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin")
	}
	if r.Method == http.MethodOptions {
		if r.Header.Get("Origin") == "" {
			http.Error(w, diagcatalog.Text(diagcatalog.ServerapiOriginIsRequired), http.StatusForbidden)
			return false
		}
		method := r.Header.Get("Access-Control-Request-Method")
		if method != "GET" && method != "POST" && method != "DELETE" {
			http.Error(w, diagcatalog.Text(diagcatalog.ServerapiMethodIsNotAllowed), http.StatusForbidden)
			return false
		}
		for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
			switch strings.ToLower(strings.TrimSpace(header)) {
			case "", "authorization", "content-type", "last-event-id", "connect-protocol-version", "connect-timeout-ms", "x-grpc-web", "grpc-timeout", "x-user-agent":
			default:
				http.Error(w, diagcatalog.Text(diagcatalog.ServerapiHeaderIsNotAllowed), http.StatusForbidden)
				return false
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Last-Event-ID, Connect-Protocol-Version, Connect-Timeout-Ms, X-Grpc-Web, Grpc-Timeout, X-User-Agent")
		w.WriteHeader(http.StatusNoContent)
		return false
	}
	// Browser WebSocket APIs cannot set Authorization. The first bounded frame
	// authenticates that transport before any command or event is accepted.
	if !t.socketHandshake(r) {
		values := r.Header.Values("Authorization")
		if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") || !t.authenticates(strings.TrimPrefix(values[0], "Bearer ")) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeAuthError(w, r)
			return false
		}
	}
	return true
}
