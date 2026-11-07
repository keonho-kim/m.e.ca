// Purpose: Expose the executing compatible conversation without replacing the native selection in snapshots.
// Pattern: Controlled model and native RPC integration test.
// Usage: go test ./internal/serverapi -run TestSnapshotSeparatesExecutingConversation.
// Related: api/meca/v1/state.proto, internal/commands/server/snapshot.go.
package serverapi

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestSnapshotSeparatesExecutingConversation(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, emit func(model.Event) error) error {
		close(started)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
			return emit(model.Event{Delta: "done", Done: true})
		}
	}), false)
	defer close(release)
	w := tlsWire(t, "grpc", f.api)
	ctx := deadline(t)
	selected := call(t, ctx, w, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
	transport := openAITransport(t, f, "sse")
	reply := openAIRequest(t, transport, "POST", "/v1/responses", `{"model":"meca","input":"background","background":true}`, "snapshot-execution")
	if reply.Code != 200 {
		t.Fatal(reply.Code, reply.Body.String())
	}
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("model did not start")
	}
	state := call(t, ctx, w, &v1.Command{Operation: &v1.Command_State{State: &v1.Empty{}}}).GetState()
	if state.GetSession().GetId() != selected.GetSession().GetId() || state.GetExecution().GetSession().GetId() == state.GetSession().GetId() || state.GetExecution().GetSession().GetId() != state.GetTask().GetSessionId() || state.GetExecution().GetModel().GetModel() == "" {
		t.Fatal("snapshot mixed selected and executing conversations", state)
	}
}
