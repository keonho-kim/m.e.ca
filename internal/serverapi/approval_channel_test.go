// Purpose: Resolve native task approvals exactly once through the shared HTTP channel on every protocol.
// Pattern: Controlled engine and authenticated HTTP integration test.
// Usage: go test -race ./internal/serverapi -run TestNativeApprovalChannel.
// Related: internal/serverapi/task_control.go, internal/commands/server/controls.go.
package serverapi

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestNativeApprovalChannel(t *testing.T) {
	for _, protocol := range []string{"sse", "socket", "grpc"} {
		t.Run(protocol, func(t *testing.T) {
			var calls atomic.Int32
			f := newFixture(t, generateFunc(func(_ context.Context, _ model.Request, emit func(model.Event) error) error {
				if calls.Add(1) > 1 {
					return emit(model.Event{Delta: "approved", Done: true})
				}
				return emit(model.Event{Calls: []tool.Call{{ID: "write", Name: "write_file", Arguments: json.RawMessage(`{"path":"approved.txt","content":"approved"}`)}}, Done: true})
			}), true)
			transport := nativeTransport(t, f, protocol)
			selected := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
			started := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: selected.GetSession().GetId(), Key: "approval-request", Text: "write"}}}).GetTask()
			if started == nil {
				t.Fatal("native task was not admitted")
			}
			cursor, approval := selected.Cursor, ""
			for approval == "" {
				batch, err := f.app.Next(deadline(t), cursor)
				if err != nil {
					t.Fatal(err)
				}
				if batch.Snapshot != nil {
					cursor = batch.Snapshot.Cursor
					for _, p := range batch.Snapshot.Approvals {
						if p.Task == started.Id {
							approval = p.ID
						}
					}
				}
				for _, e := range batch.Events {
					cursor = e.Cursor
					if e.Task == started.Id && e.Record.Kind == "approval_required" {
						approval = e.ApprovalID
					}
				}
			}
			if f.executed.Load() != 0 {
				t.Fatal("effect ran before approval")
			}
			pending := nativeRequest(t, transport, "GET", "/v1/tasks/"+started.Id+"/approvals", "")
			var state v1.Reply
			if err := protojson.Unmarshal(pending.Body.Bytes(), &state); err != nil || len(state.GetState().GetApprovals()) != 1 || state.GetState().Approvals[0].TaskId != started.Id {
				t.Fatal(pending.Code, pending.Body.String(), err)
			}
			path := "/v1/tasks/" + started.Id + "/approvals/" + approval
			if invalid := nativeRequest(t, transport, "POST", path, `{"allow":false,"allow":true}`); invalid.Code != 400 {
				t.Fatal("ambiguous approval accepted", invalid.Code)
			}
			if decided := nativeRequest(t, transport, "POST", path, `{"allow":true}`); decided.Code != 200 {
				t.Fatal(decided.Code, decided.Body.String())
			}
			if stale := nativeRequest(t, transport, "POST", path, `{"allow":true}`); stale.Code != 409 {
				t.Fatal("stale approval accepted", stale.Code)
			}
			task := waitNativeTask(t, deadline(t), f.app, started.Id, cursor)
			if task.Status != "answered" || f.executed.Load() != 1 {
				t.Fatal(task, f.executed.Load())
			}
		})
	}
}
