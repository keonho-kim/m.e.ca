// Purpose: Reproduce whether live decoded socket commands retain their input admission.
// Pattern: network and resource ownership regression test.
// Usage: go test ./internal/serverapi.
// Related: internal/serverapi/socket.go, internal/serverapi/commands.go, internal/serverapi/admission/input.go.
package serverapi

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type socketWaitingLedger struct{ entered chan struct{} }

func (l socketWaitingLedger) Admit(context.Context, string, string, [32]byte) (server.Task, bool, error) {
	return server.Task{}, false, nil
}
func (l socketWaitingLedger) Get(ctx context.Context, _ string) (server.Task, error) {
	close(l.entered)
	<-ctx.Done()
	return server.Task{}, ctx.Err()
}
func (socketWaitingLedger) Finish(context.Context, server.Task) error { return nil }

func TestSocketRetainsDecodedInputAdmission(t *testing.T) {
	ctx := deadline(t)
	const session = "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"
	ledger := socketWaitingLedger{entered: make(chan struct{})}
	app, err := server.New(ctx, server.Options{
		Ledger: ledger,
		Open: func(context.Context, string, bool, config.Config) (*server.Conversation, error) {
			return &server.Conversation{Runtime: agentrequest.Runtime{Session: sessionstate.Session{ID: session}}, Close: func() error { return nil }}, nil
		},
		List: func(context.Context, string, int) ([]sessionstate.Conversation, string, error) { return nil, "", nil },
		PrepareModel: func(context.Context, connection.Selection, string, setup.Defaults) (server.PreparedModel, error) {
			return server.PreparedModel{}, nil
		},
		UpdateConfig: func(context.Context, func(*config.Config)) (config.Config, error) { return config.Config{}, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if _, err := app.Select(ctx, "", true); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(nil)
	transport, err := newTransport(ctx, &API{Service: app, Redact: func(s string) string { return s }}, "socket", testToken, []string{srv.Listener.Addr().String()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	srv.Config.Handler = transport
	srv.Start()
	defer srv.Close()
	defer transport.Close()
	conn, _, err := websocket.Dial(ctx, strings.Replace(srv.URL, "http:", "ws:", 1)+"/v1/socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"authenticate":"`+testToken+`"}`)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := conn.Read(ctx); err != nil {
		t.Fatal(err)
	}
	frame := `{"command":{"id":"large","start":{"sessionId":"` + session + `","key":"review-key","text":"` + strings.Repeat("x", 8<<20) + `"}}}`
	if err := conn.Write(ctx, websocket.MessageText, []byte(frame)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ledger.entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	release, available := transport.inputs.Try(25 << 20)
	if available {
		release()
		t.Fatal("decoded 8 MiB command was undercharged while it waits in ledger I/O")
	}
	release, available = transport.inputs.Try(20 << 20)
	if !available {
		t.Fatal("small command occupied the whole input budget")
	}
	release()
}
