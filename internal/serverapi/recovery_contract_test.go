// Purpose: Measure valid concurrent draft snapshot against the configured RPC limit.
// Pattern: Wire-size and request recovery regression tests.
// Usage: go test ./internal/serverapi -run TestRegression.
// Related: internal/serverapi/rpc.go, internal/serverapi/projection.go.
package serverapi

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/commands/server"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/proto"
	"strings"
	"testing"
)

func TestRegressionConcurrentSnapshotFitsRPC(t *testing.T) {
	value := server.Snapshot{Concurrency: 33}
	text := strings.Repeat("x", 128<<10)
	for i := 0; i < 33; i++ {
		id := fmt.Sprintf("task-%d", i)
		value.Tasks = append(value.Tasks, server.Task{ID: id, Status: "running"})
		value.Drafts = append(value.Drafts, server.Draft{Task: id, Text: text, Thinking: text})
	}
	size := proto.Size(snapshot(value))
	if size > 8<<20 {
		t.Fatalf("valid 33-task snapshot = %d bytes; RPC send limit = %d", size, 8<<20)
	}
}

func TestRegressionImageRetryAfterRelease(t *testing.T) {
	f := newFixture(t, visionGenerate{func(_ context.Context, _ model.Request, emit func(model.Event) error) error {
		return emit(model.Event{Delta: "done", Done: true})
	}}, false)
	transport := nativeTransport(t, f, "sse")
	image := uploadImage(t, transport, nativePNG(t))
	selected := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
	command := &v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: selected.GetSession().GetId(), Key: "audit-image-retry", AttachmentIds: []string{image.ID}}}}
	first := nativeCommand(t, transport, command)
	if first.GetTask() == nil {
		t.Fatal(first)
	}
	waitNativeTask(t, deadline(t), f.app, first.GetTask().GetId(), selected.Cursor)
	if err := f.app.ReleaseImage(image.ID); err != nil {
		t.Fatal(err)
	}
	retry := nativeCommand(t, transport, command)
	if retry.GetTask().GetId() != first.GetTask().GetId() {
		t.Fatalf("same admitted request cannot be recovered after upload release: %v", retry.GetError())
	}
}
