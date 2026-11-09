// Purpose: Keep task control responsive when general HTTP or original-read capacity is occupied.
// Pattern: Deterministic transport admission regression tests.
// Usage: go test -race ./internal/serverapi -run TestReservedControl.
// Related: internal/serverapi/transport.go, internal/serverapi/commands.go.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"testing"
)

func TestReservedControlAndReadAdmission(t *testing.T) {
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, _ func(model.Event) error) error {
		<-ctx.Done()
		return ctx.Err()
	}), false)
	transport := nativeTransport(t, f, "sse")
	task, cursor := startNativeTask(t, f, "reserved-control", "hold")
	for range maxRequests {
		transport.requests <- struct{}{}
	}
	response := nativeRequest(t, transport, "POST", "/v1/tasks/"+task.ID+"/cancel", "")
	if response.Code != 200 {
		t.Fatal("control shared ordinary request capacity", response.Code, response.Body.String())
	}
	for range maxRequests {
		<-transport.requests
	}
	waitNativeTask(t, deadline(t), f.app, task.ID, cursor)
	release, ok := f.api.readLimit().Try(1)
	if !ok {
		t.Fatal("read capacity unexpectedly busy")
	}
	defer release()
	reply := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_History{History: &v1.HistoryRequest{SessionId: task.Session}}})
	if reply.GetError().GetCode() != "resource_exhausted" {
		t.Fatal("expensive read waited instead of rejecting", reply)
	}
	resources := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Resources{Resources: &v1.Empty{}}})
	if resources.GetResources().GetResidentBytes() == 0 && resources.GetResources().GetErrorCode() == "" {
		t.Fatal(resources)
	}
}
