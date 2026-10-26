// Purpose: Verify durable compaction counters, atomic replacement and step/file evidence projection.
// Pattern: Repository behavior tests.
// Usage: go test ./internal/storage/sqlite -run TestMemory.
// Related: internal/storage/sqlite/memory.go, internal/storage/sqlite/memory_sources.go.
package sqlite

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestMemoryCountersAndOriginalHistory(t *testing.T) {
	ctx := context.Background()
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	session := store.Session()
	if err = store.AppendMessage(ctx, session.ID, modelapi.Message{Role: "user", Content: "goal"}); err != nil {
		t.Fatal(err)
	}
	if err = store.AddContextTokens(ctx, session.ID, compact.TokenCount{Amount: 256000, Kind: "request"}); err != nil {
		t.Fatal(err)
	}
	mark, err := store.MemoryWatermark(ctx, session.ID, 0)
	if err != nil || mark.Tokens != 256000 {
		t.Fatalf("counter: %+v %v", mark, err)
	}
	memory := compact.Memory{Revision: 1, Through: mark.Timeline, UsageThrough: mark.Usage, Markdown: "## Requests\n- goal"}
	if err = store.InstallMemory(ctx, session.ID, 0, memory); err != nil {
		t.Fatal(err)
	}
	if err = store.InstallMemory(ctx, session.ID, 0, memory); err == nil {
		t.Fatal("stale memory replaced committed revision")
	}
	mark, err = store.MemoryWatermark(ctx, session.ID, memory.UsageThrough)
	if err != nil || mark.Tokens != 0 {
		t.Fatalf("watermark not reset: %+v %v", mark, err)
	}
	history, err := store.Messages(ctx, session.ID, 10)
	if err != nil || len(history) != 1 || history[0].Content != "goal" {
		t.Fatal("original history lost")
	}
	if err = store.AddContextTokens(ctx, session.ID, compact.EstimateBytes(3)); err != nil {
		t.Fatal(err)
	}
	mark, err = store.MemoryWatermark(ctx, session.ID, memory.UsageThrough)
	if err != nil || mark.Tokens != 1 || !mark.Estimated {
		t.Fatalf("new usage: %+v %v", mark, err)
	}
}
func TestMemoryProjectsStepChanges(t *testing.T) {
	ctx := context.Background()
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	session := store.Session()

	cp := runrecord.Checkpoint{ID: "task", Objective: "goal", Status: runrecord.Running, Phase: "execution", InstructionsDigest: strings.Repeat("a", 64)}
	if err = store.RecordEvent(ctx, session.ID, runrecord.Event{Kind: "run_checkpoint", Checkpoint: &cp}); err != nil {
		t.Fatal(err)
	}
	op := toolapi.Operation{RunID: "task", CallID: "call", Name: "write_file", Status: "tool_succeeded", Changes: []toolapi.FileChange{{Action: "write_file", Path: "src/session.go"}}}
	if err = store.RecordEvent(ctx, session.ID, runrecord.Event{Kind: "tool_succeeded", Operation: &op}); err != nil {
		t.Fatal(err)
	}
	mark, err := store.MemoryWatermark(ctx, session.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	records, _, err := store.MemoryRecords(ctx, session.ID, 0, mark.Timeline)
	if err != nil || len(records) != 1 {
		t.Fatalf("projection: %+v %v", records, err)
	}
	if !strings.Contains(records[0].Text, "src/session.go") {
		t.Fatal("step changed-file facts lost")
	}
}

func TestMemoryUsesLatestRequestContextInsteadOfCumulativeCost(t *testing.T) {
	ctx := context.Background()
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	id := store.Session().ID
	// Old counter facts remain stored, but are not input+output request usage.
	if _, err = store.db.ExecContext(ctx, "INSERT INTO conversation_tokens(session_id,amount,estimated,kind) VALUES(?,900000,1,'model')", id); err != nil {
		t.Fatal(err)
	}
	for _, count := range []compact.TokenCount{{Amount: 199999, Kind: "request"}, {Amount: 1, Kind: "request"}} {
		if err = store.AddContextTokens(ctx, id, count); err != nil {
			t.Fatal(err)
		}
	}
	mark, err := store.MemoryWatermark(ctx, id, 0)
	if err != nil || mark.Tokens != 1 || mark.Estimated {
		t.Fatalf("%+v %v", mark, err)
	}
	next, err := store.MemoryWatermark(ctx, id, mark.Usage)
	if err != nil || next.Tokens != 0 {
		t.Fatalf("%+v %v", next, err)
	}
}
