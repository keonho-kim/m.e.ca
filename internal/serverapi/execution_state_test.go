// Purpose: Keep native snapshot selection separate from detached execution state.
// Pattern: Controlled engine and native RPC contract test.
// Usage: go test ./internal/serverapi -run TestSnapshotSeparatesExecutingConversation.
// Related: api/meca/v1/state.proto, internal/commands/server/snapshot.go.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"testing"
)

func TestSnapshotSeparatesExecutingConversation(t *testing.T) {
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, _ func(model.Event) error) error {
		<-ctx.Done()
		return ctx.Err()
	}), false)
	task, _ := startNativeTask(t, f, "snapshot-execution", "wait")
	selected, err := f.app.Select(deadline(t), "", true)
	if err != nil {
		t.Fatal(err)
	}
	wire := tlsWire(t, "grpc", f.api)
	state := call(t, deadline(t), wire, &v1.Command{Operation: &v1.Command_State{State: &v1.StateRequest{}}}).GetState()
	if state.GetSession().GetId() != selected.Session.ID || state.GetExecution().GetSession().GetId() != task.Session || state.GetExecution().GetTaskId() != task.ID || len(state.Tasks) != 1 {
		t.Fatal("snapshot mixed selection and execution", state)
	}
}
