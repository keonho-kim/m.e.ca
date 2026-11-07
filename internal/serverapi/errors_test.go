// Purpose: Classify malformed session identities and absent detail records as public client errors.
// Pattern: API boundary regression tests using real SQLite storage.
// Usage: go test ./internal/serverapi -run TestCommandError.
// Related: internal/serverapi/commands.go, internal/commands/server/detail.go.
package serverapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestCommandErrorClassification(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	ctx := deadline(t)
	if _, err := f.app.Select(ctx, "", true); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, code string
		command    *v1.Command
	}{
		{"malformed session", "invalid_argument", &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{SessionId: "not-a-session"}}}},
		{"missing session", "not_found", &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{SessionId: "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"}}}},
		{"missing tool detail", "not_found", &v1.Command{Operation: &v1.Command_Detail{Detail: &v1.DetailRequest{Source: &v1.DetailRequest_Tool{Tool: &v1.EventReference{Sequence: 42}}}}}},
		{"missing source session", "not_found", &v1.Command{Operation: &v1.Command_Detail{Detail: &v1.DetailRequest{Source: &v1.DetailRequest_Tool{Tool: &v1.EventReference{SessionId: "session_AAAAAAAAAAAAAAAAAAAAAAAAAA", Sequence: 42}}}}}},
		{"invalid model", "invalid_argument", &v1.Command{Operation: &v1.Command_SetModel{SetModel: &v1.ModelRequest{Provider: "unknown", Model: "fixture"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reply, err := f.api.Call(ctx, tc.command)
			if err != nil || reply.GetError().GetCode() != tc.code {
				t.Fatal(reply, err)
			}
		})
	}
}

func TestCanceledReadReturnsHTTPConflict(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport, err := newTransport(t.Context(), f.api, "sse", testToken, []string{"localhost:8787"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.Close()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:8787/v1/command", strings.NewReader(`{"history":{}}`))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	reply := httptest.NewRecorder()
	transport.ServeHTTP(reply, req)
	if reply.Code != http.StatusConflict || !strings.Contains(reply.Body.String(), `"canceled"`) {
		t.Fatal(reply.Code, reply.Body.String())
	}
}
