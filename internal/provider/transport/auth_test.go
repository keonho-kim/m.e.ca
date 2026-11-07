// Purpose: Reject cleartext remote API keys and cross-origin authenticated redirects.
// Pattern: Credential transport boundary tests.
// Usage: go test ./internal/provider/transport -run TestAPIKey.
// Related: internal/provider/transport/transport.go, internal/provider/transport/client.go.
package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestAPIKeyRequiresTLSOutsideLoopback(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenAI, connection.Claude, connection.Gemini} {
		c := New("secret", kind)
		for _, endpoint := range []string{"http://example.com/v1", "http://192.0.2.1/v1"} {
			if _, err := c.NewRequest(t.Context(), "GET", endpoint, nil); err == nil {
				t.Fatal("API key permitted over cleartext remote HTTP", kind)
			}
		}
		for _, endpoint := range []string{"http://127.0.0.1/v1", "http://[::1]/v1", "https://example.com/v1"} {
			if _, err := c.NewRequest(t.Context(), "GET", endpoint, nil); err != nil {
				t.Fatal(endpoint, err)
			}
		}
		c.Close()
	}
}

func TestAPIKeyHTTPWhitespaceCannotBypassRedaction(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenAI, connection.Claude, connection.Gemini} {
		raw := " \tprovider-secret\t "
		vault := &credentials.Vault{}
		vault.Remember(raw)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actual := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if kind == connection.Claude {
				actual = r.Header.Get("x-api-key")
			}
			if kind == connection.Gemini {
				actual = r.Header.Get("x-goog-api-key")
			}
			if actual != "provider-secret" {
				t.Error("wire credential normalization differs")
			}
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "rejected " + actual}})
		}))
		client := New(raw, kind)
		request, err := client.NewRequest(t.Context(), http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.SendOnce(request, time.Second)
		client.Close()
		server.Close()
		if err == nil || strings.Contains(vault.Redact(client.FailureReason(err)), "provider-secret") {
			t.Fatal("normalized credential leaked in provider error", err)
		}
	}
}

func TestAPIKeyCannotFollowAnotherOrigin(t *testing.T) {
	var called atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called.Store(true) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer redirect.Close()
	c := New("secret", connection.Claude)
	defer c.Close()
	req, err := c.NewRequest(context.Background(), "GET", redirect.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := c.HTTP.Do(req)
	if response != nil {
		response.Body.Close()
	}
	if err == nil || called.Load() {
		t.Fatal("authenticated redirect crossed origin", err)
	}
}
