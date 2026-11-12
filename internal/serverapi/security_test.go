// Purpose: Verify authentication, browser trust checks and private token-file handling.
// Pattern: Security boundary integration tests.
// Usage: go test ./internal/serverapi -run 'TestSecurity|TestToken|TestSocketAuthentication'.
// Related: internal/serverapi/security.go, internal/serverapi/token.go.
package serverapi

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/coder/websocket"
	"github.com/keonho-kim/meca/internal/agent/model"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/serverapi/gen/mecav1/mecav1connect"
)

func idleGenerator(_ context.Context, _ model.Request, emit func(model.Event) error) error {
	return emit(model.Event{Delta: "done", Done: true})
}

func TestSecurityBeforeControlDispatch(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	w := tlsWire(t, "sse", f.api)
	for _, tc := range []struct {
		name, token, host, origin, method, path string
		status                                  int
	}{
		{name: "missing token", status: 401},
		{name: "wrong token", token: "wrong", status: 401},
		{name: "rebound host", token: testToken, host: "evil.example:443", status: 403},
		{name: "untrusted origin", token: testToken, origin: "https://evil.example", status: 403},
		{name: "null origin", token: testToken, origin: "null", status: 403},
		{name: "allowed origin", token: testToken, origin: "https://frontend.example", status: 200},
		{name: "token in URL", token: testToken, path: "/v1/command?token=secret", status: 400},
		{name: "preflight", origin: "https://frontend.example", method: "OPTIONS", status: 204},
		{name: "untrusted preflight", origin: "https://evil.example", method: "OPTIONS", status: 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			method, path := tc.method, tc.path
			if method == "" {
				method = "POST"
			}
			if path == "" {
				path = "/v1/command"
			}
			req, err := http.NewRequestWithContext(deadline(t), method, w.url+path, strings.NewReader(`{"state":{}}`))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			if tc.host != "" {
				req.Host = tc.host
			}
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if method == "OPTIONS" {
				req.Header.Set("Access-Control-Request-Method", "POST")
				req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
			}
			res, err := w.http.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != tc.status {
				t.Fatal(res.Status)
			}
			if tc.origin == "https://frontend.example" && res.Header.Get("Access-Control-Allow-Origin") != tc.origin {
				t.Fatal("allowed origin missing")
			}
		})
	}
}

func TestSecurityGRPCAuthenticationStatus(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	w := tlsWire(t, "grpc", f.api)
	rpc := mecav1connect.NewMecaServiceClient(connect.NewClient(connecthttp.NewTransport(w.http, w.url, connecthttp.WithGRPC())))
	_, err := rpc.Call(deadline(t), &v1.Command{Operation: &v1.Command_State{State: &v1.StateRequest{}}})
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatal(err)
	}
}

func TestSocketAuthenticationPrecedesCommands(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	w := tlsWire(t, "socket", f.api)
	ctx := deadline(t)
	conn, _, err := websocket.Dial(ctx, strings.Replace(w.url, "https:", "wss:", 1)+"/v1/socket", &websocket.DialOptions{HTTPClient: w.http})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"command":{"id":"x","selectSession":{"create":true}}}`)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := conn.Read(ctx); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatal(err)
	}
	if f.app.State().Session.ID != "" {
		t.Fatal("unauthenticated command changed state")
	}
}

func TestTokenPrivateAtomicAndRotates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.token")
	first := NewToken()
	if err := PublishToken(path, first); err != nil || !validToken(first) {
		t.Fatal(err)
	}
	again := NewToken()
	if err := PublishToken(path, again); err != nil || again == first {
		t.Fatal("server startup reused its previous token", err)
	}
	stored, err := os.ReadFile(path)
	if err != nil || strings.TrimSuffix(string(stored), "\n") != again {
		t.Fatal("published credential does not match this startup", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("published credential is not private", err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if err := PublishToken(path, NewToken()); err == nil {
		t.Fatal("public token accepted")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if err := PublishToken(link, NewToken()); err == nil {
		t.Fatal("token symlink accepted")
	}
}

func TestSecurityExactAuthoritiesIncludeDefaultPorts(t *testing.T) {
	for _, host := range []string{"frontend.example", "frontend.example:443", "127.0.0.1:8787", "[::1]:8787"} {
		if !validHost(host) {
			t.Fatal("valid authority rejected", host)
		}
	}
	for _, host := range []string{"", "*.example", "https://example", "user@example", "example:", "example:99999", "example:bad", "example/path"} {
		if validHost(host) {
			t.Fatal("invalid authority accepted", host)
		}
	}
}
