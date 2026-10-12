// Purpose: Verify durable compaction counters, atomic replacement and step/file evidence projection.
// Pattern: Repository behavior tests.
// Usage: go test ./internal/storage/sqlite -run TestMemory.
// Related: internal/storage/sqlite/memory.go, internal/storage/sqlite/memory_sources.go.
package sqlite

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryCountersAndOriginalHistory(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "v.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	root := t.TempDir()
	session, err := store.RestoreSession(ctx, root, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.AppendMessage(ctx, session.ID, agent.Message{Role: "user", Content: "goal"}); err != nil {
		t.Fatal(err)
	}
	if err = store.AddContextTokens(ctx, session.ID, compact.TokenCount{Amount: 256000, Kind: "model"}); err != nil {
		t.Fatal(err)
	}
	mark, err := store.MemoryWatermark(ctx, session.ID, 0)
	if err != nil || mark.Tokens != 256000 {
		t.Fatalf("counter: %+v %v", mark, err)
	}
	memory := compact.Memory{Revision: 1, Through: mark.Timeline, UsageThrough: mark.Usage, Markdown: "## Goals\n- goal"}
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
	if err = store.AddContextTokens(ctx, session.ID, compact.Estimate("new", "user")); err != nil {
		t.Fatal(err)
	}
	mark, err = store.MemoryWatermark(ctx, session.ID, memory.UsageThrough)
	if err != nil || mark.Tokens != 3 || !mark.Estimated {
		t.Fatalf("new usage: %+v %v", mark, err)
	}
}
func TestMemoryProjectsStepChanges(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "v.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	root := t.TempDir()
	session, err := store.RestoreSession(ctx, root, root, false)
	if err != nil {
		t.Fatal(err)
	}
	step := &compact.StepRef{PlanID: "plan", Revision: 2, Index: 1, Description: "edit auth"}
	cp := run.Checkpoint{ID: "task", Objective: "goal", Status: run.Running, Phase: "execution", InstructionsDigest: strings.Repeat("a", 64), Step: step}
	if err = store.RecordEvent(ctx, session.ID, run.Event{Kind: "task_checkpoint", Checkpoint: &cp}); err != nil {
		t.Fatal(err)
	}
	op := compact.Operation{TaskID: "task", CallID: "call", Name: "move_file", Status: "tool_succeeded", Step: step, Changes: []agent.FileChange{{Action: "move_file", Path: "src/auth.go", Destination: "src/session.go"}}}
	if err = store.RecordEvent(ctx, session.ID, run.Event{Kind: "tool_succeeded", Operation: &op}); err != nil {
		t.Fatal(err)
	}
	mark, err := store.MemoryWatermark(ctx, session.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	records, _, err := store.MemoryRecords(ctx, session.ID, 0, mark.Timeline)
	if err != nil || len(records) != 2 {
		t.Fatalf("projection: %+v %v", records, err)
	}
	if records[1].Operation.Changes[0].Destination != "src/session.go" || records[0].Task.Step.Index != 1 {
		t.Fatal("step changed-file facts lost")
	}
}
