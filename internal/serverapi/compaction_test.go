// Purpose: Verify explicit compaction on a short native conversation and its reuse on the next turn.
// Pattern: SQLite and admitted-task integration test with a controlled provider.
// Usage: go test ./internal/serverapi -run TestManualCompaction.
// Related: internal/commands/server/execution.go, internal/agent/run/manual_compaction.go.
package serverapi

import (
	"context"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestManualCompaction(t *testing.T) {
	const original = "Use the blue file."
	const summary = "The selected file is blue."
	packets := make(chan model.Request, 3)
	decisions := make(chan model.Request, 3)
	release := make(chan struct{})
	compactionPrompt := prompt.Defaults().Compaction
	f := newFixture(t, generateFunc(func(ctx context.Context, request model.Request, emit func(model.Event) error) error {
		if request.Messages[0].Content == compactionPrompt {
			packets <- request
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-release:
				return emit(model.Event{Done: true, Delta: summary, Usage: &model.TokenUsage{Input: 40, Output: 5}})
			}
		}
		decisions <- request
		return emit(model.Event{Done: true, Delta: "Understood."})
	}), false, 2)
	ctx := deadline(t)
	transport := nativeTransport(t, f, "sse")
	state, err := f.app.Select(ctx, "", true)
	if err != nil {
		t.Fatal(err)
	}
	session := state.Session.ID
	start := func(key, text string, paths ...string) *v1.Reply {
		return nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{
			SessionId: session, Key: key, Text: text, Paths: paths,
		}}})
	}
	wait := func(reply *v1.Reply, outcome string) {
		t.Helper()
		if reply.GetTask() == nil {
			t.Fatal(reply)
		}
		task := waitNativeTask(t, ctx, f.app, reply.GetTask().Id, state.Cursor)
		if task.Outcome != outcome {
			t.Fatalf("task outcome = %q, want %q (%s)", task.Outcome, outcome, task.Error)
		}
	}
	wait(start("seed-history", original), "completed")
	<-decisions
	checkpoint, err := f.store.LatestRun(ctx, session)
	if err != nil || checkpoint == nil {
		t.Fatal(checkpoint, err)
	}
	if reply := start("bad-compact", "!compact", "blue.txt"); reply.GetError().GetCode() != "invalid_argument" {
		t.Fatal("compaction accepted a file reference", reply)
	}
	if _, err := f.app.Configure(ctx, session, nil, new(true)); err != nil {
		t.Fatal(err)
	}
	awaitSummary := func() model.Request {
		t.Helper()
		select {
		case packet := <-packets:
			if len(packet.Tools) != 0 || !strings.Contains(packet.Messages[1].Content, original) {
				t.Fatal("unexpected summary packet", packet)
			}
			return packet
		case <-ctx.Done():
			t.Fatal(ctx.Err())
			return model.Request{}
		}
	}
	first := start("cancel-compact", "!compact")
	awaitSummary()
	id := first.GetTask().GetId()
	if duplicate := start("cancel-compact", "!compact"); duplicate.GetTask().GetId() != id {
		t.Fatal("compaction request was not idempotent", duplicate)
	}
	if concurrent := start("busy-compact", "!compact"); concurrent.GetError().GetCode() != "conflict" {
		t.Fatal("a second request accessed the active conversation", concurrent)
	}
	if err := f.app.Steer(id, "follow-up"); err == nil {
		t.Fatal("compaction accepted a direction change")
	}
	if err := f.app.Cancel(id); err != nil {
		t.Fatal(err)
	}
	wait(first, "interrupted")
	if memory, err := f.store.Memory(ctx, session); err != nil || memory.Revision != 0 {
		t.Fatal("canceled compaction installed a summary", memory, err)
	}

	second := start("apply-compact", "!compact")
	awaitSummary()
	close(release)
	wait(second, "completed")
	if memory, err := f.store.Memory(ctx, session); err != nil || memory.Revision != 1 || memory.Markdown != summary {
		t.Fatal("summary missing", memory, err)
	}
	if latest, err := f.store.LatestRun(ctx, session); err != nil || latest == nil || latest.ID != checkpoint.ID {
		t.Fatal("compaction replaced the real conversation checkpoint", latest, err)
	}
	raw, err := f.store.ContextMessages(ctx, session, 0, 64<<10, false, nil)
	if err != nil || len(raw) != 2 || raw[0].Content != original || raw[1].Content != "Understood." {
		t.Fatal("compaction altered original messages", raw, err)
	}
	wait(start("empty-compact", "!compact"), "failed")
	if len(packets) != 0 {
		t.Fatal("repeated compaction called the model without new history")
	}
	if _, err := f.app.Configure(ctx, session, nil, new(false)); err != nil {
		t.Fatal(err)
	}
	wait(start("next-request", "Where is the file?"), "completed")
	next := <-decisions
	found := false
	for _, message := range next.Messages {
		found = found || strings.Contains(message.Content, summary)
		if message.Role == "user" && message.Content == original {
			t.Fatal("compacted text was replayed in raw history")
		}
	}
	if !found {
		t.Fatal("next request did not see the installed summary")
	}
}
