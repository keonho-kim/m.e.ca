// Purpose: Exercise native HTTP control, task completion and authenticated protocol fixtures.
// Pattern: Test boundary helpers over the production native API.
// Usage: Native concurrency, approval, image and route-removal integration tests share these helpers.
// Related: internal/serverapi/fixture_test.go, internal/serverapi/http.go.
package serverapi

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
)

func nativeTransport(t *testing.T, f *fixture, protocol string) *Transport {
	t.Helper()
	transport, err := newTransport(t.Context(), f.api, protocol, testToken, []string{"localhost:8787"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(transport.Close)
	return transport
}
func nativeRequest(t *testing.T, transport *Transport, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(deadline(t), method, "http://localhost:8787"+path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+testToken)
	r.Header.Set("Content-Type", "application/json")
	reply := httptest.NewRecorder()
	transport.ServeHTTP(reply, r)
	return reply
}
func nativeCommand(t *testing.T, transport *Transport, command *v1.Command) *v1.Reply {
	t.Helper()
	body, err := protojson.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	response := nativeRequest(t, transport, "POST", "/v1/command", string(body))
	var reply v1.Reply
	if err := protojson.Unmarshal(response.Body.Bytes(), &reply); err != nil {
		t.Fatal(response.Code, response.Body.String(), err)
	}
	return &reply
}
func startNativeTask(t *testing.T, f *fixture, key, text string) (server.Task, string) {
	t.Helper()
	state, err := f.app.Select(deadline(t), "", true)
	if err != nil {
		t.Fatal(err)
	}
	task, err := f.app.Start(deadline(t), state.Session.ID, key, text)
	if err != nil {
		t.Fatal(err)
	}
	return task, state.Cursor
}
func waitNativeTask(t *testing.T, ctx context.Context, app *server.Service, id, cursor string) server.Task {
	t.Helper()
	for {
		batch, err := app.Next(ctx, cursor)
		if err != nil {
			t.Fatal(err)
		}
		if batch.Snapshot != nil {
			cursor = batch.Snapshot.Cursor
			task, err := app.TaskStatus(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if task.Status != "running" && task.Status != "canceling" && task.Status != "accepted" {
				return task
			}
		}
		for _, event := range batch.Events {
			cursor = event.Cursor
			if event.Task == id && event.Record.Kind == "request_finished" {
				task, err := app.TaskStatus(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				return task
			}
		}
	}
}
