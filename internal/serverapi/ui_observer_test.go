// Purpose: Verify browser-only multiplexing and authentication lifetime on the default SSE server.
// Pattern: Real HTTP/WebSocket contract with bounded waits and joined fixture shutdown.
// Usage: go test -race ./internal/serverapi -run TestUIObserver.
// Related: internal/serverapi/ui_observer.go, internal/serverapi/browser_sessions.go.
package serverapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestUIObserverSharesSocketAndRevokesWithCookie(t *testing.T) {
	f := newFixture(t, nil, false, 1)
	transport := nativeTransport(t, f, "sse")
	if err := transport.enableUI(http.NotFoundHandler()); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(transport)
	defer server.Close()
	address, _ := url.Parse(server.URL)
	transport.hosts[address.Host] = true
	request, _ := http.NewRequestWithContext(t.Context(), "POST", server.URL+"/v1/ui/login", nil)
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Origin", server.URL)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatal(response.StatusCode)
	}
	cookie := response.Cookies()[0]
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	headers := make(http.Header)
	headers.Set("Origin", server.URL)
	headers.Set("Cookie", cookie.String())
	conn, _, err := websocket.Dial(ctx, "ws://"+address.Host+"/v1/ui/events", &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	for id, target := range map[string]string{"conversation": "/v1/events", "files": "/v1/files/events?path=" + url.QueryEscape(f.project)} {
		body, _ := json.Marshal(uiWatchFrame{ID: id, URL: target})
		if err := conn.Write(ctx, websocket.MessageText, body); err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	for len(seen) < 2 {
		_, body, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var frame struct {
			ID       string
			Messages []uiMessage
		}
		if err := json.Unmarshal(body, &frame); err != nil {
			t.Fatal(err)
		}
		if len(frame.Messages) == 0 || frame.Messages[0].Event == "error" {
			t.Fatal("missing observer data", string(body))
		}
		seen[frame.ID] = true
	}
	probe, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/v1/ui/session", nil)
	probe.AddCookie(cookie)
	response, err = server.Client().Do(probe)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal(response.StatusCode)
	}
	logout, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/v1/ui/logout", strings.NewReader(""))
	logout.AddCookie(cookie)
	logout.Header.Set("Origin", server.URL)
	logout.Header.Set("X-Meca-UI", "1")
	response, err = server.Client().Do(logout)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 204 {
		t.Fatal(response.StatusCode)
	}
	if _, _, err := conn.Read(ctx); err == nil {
		t.Fatal("revoked browser observer survived logout")
	}
}
