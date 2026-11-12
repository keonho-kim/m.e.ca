// Purpose: Deliver every live team peer and resume the native event stream without stopping the task.
// Pattern: Real team engine and multi-transport observer lifecycle integration test.
// Usage: go test -race ./internal/serverapi -run TestTeamFullStream.
// Related: internal/serverapi/watch.go, internal/agent/team/events.go, internal/serverapi/team_fixture_test.go.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	"sync/atomic"
	"testing"
)

func TestTeamFullStreamIncludesPeersImagesAndResume(t *testing.T) {
	for _, protocol := range []string{"sse", "socket", "grpc"} {
		t.Run(protocol, func(t *testing.T) {
			var images atomic.Int32
			f := newFixture(t, visionGenerate{func(ctx context.Context, r model.Request, emit func(model.Event) error) error {
				for _, m := range r.Messages {
					if len(m.Images) == 1 {
						images.Add(1)
						break
					}
				}
				if err := emit(model.Event{Delta: "live public contribution", Thinking: "private-thinking-canary", Streaming: true}); err != nil {
					return err
				}
				<-ctx.Done()
				return ctx.Err()
			}}, false)
			ctx := deadline(t)
			state, err := f.app.Select(ctx, "", true)
			if err != nil {
				t.Fatal(err)
			}
			enabled := true
			if _, err := f.app.Configure(ctx, f.app.State().Session.ID, nil, &enabled); err != nil {
				t.Fatal(err)
			}
			transport := nativeTransport(t, f, protocol)
			image := uploadImage(t, transport, nativePNG(t))
			pixels, err := f.app.ImageAttachments([]string{image.ID})
			if err != nil {
				t.Fatal(err)
			}
			task, err := f.app.Start(ctx, state.Session.ID, "team-image-stream", "collaborate", pixels...)
			if err != nil {
				t.Fatal(err)
			}
			wire := tlsWire(t, protocol, f.api)
			stream, err := wire.watch(ctx, state.Cursor)
			if err != nil {
				t.Fatal(err)
			}
			agents := map[string]bool{}
			cursor := state.Cursor
			for len(agents) < 2 {
				n := next(t, stream)
				event := n.GetEvent()
				if event == nil {
					continue
				}
				if event.GetMessage().GetThinking() != "" {
					t.Fatal("private team reasoning exposed")
				}
				if event.Cursor != "" {
					cursor = event.Cursor
				}
				if event.TaskId == task.ID && event.Kind == "assistant_delta" && event.Agent != "" && event.GetMessage().GetText() != "" {
					agents[event.Agent] = true
				}
			}
			if images.Load() != 2 {
				t.Fatal("team image snapshots were not delivered", images.Load())
			}
			if err := stream.close(); err != nil {
				t.Fatal(err)
			}
			if status, err := f.app.TaskStatus(ctx, task.ID); err != nil || status.Status != "running" {
				t.Fatal("stream pause stopped the task", status, err)
			}
			again, err := wire.watch(ctx, cursor)
			if err != nil {
				t.Fatal(err)
			}
			defer again.close()
			if err := f.app.Steer(task.ID, "continue after observer resume"); err != nil {
				t.Fatal(err)
			}
			for {
				n := next(t, again)
				event := n.GetEvent()
				if event == nil || event.TaskId != task.ID {
					continue
				}
				if event.Kind == "user_message" && event.GetMessage().GetText() == "continue after observer resume" {
					break
				}
			}
			decision := nativeRequest(t, transport, "POST", "/v1/tasks/"+task.ID+"/cancel", "")
			if decision.Code != 200 {
				t.Fatal(decision.Code, decision.Body.String())
			}
			waitNativeTask(t, ctx, f.app, task.ID, cursor)
		})
	}
}
