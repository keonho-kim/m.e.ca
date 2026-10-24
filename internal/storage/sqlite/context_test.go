// Purpose: Verify complete context ordering, committed summary cutoffs and lossless budget failures.
// Pattern: Repository behavior tests.
// Usage: go test ./internal/storage/sqlite -run TestContextMessages.
// Related: internal/storage/sqlite/context.go, internal/storage/sqlite/timeline.go.
package sqlite

import (
	"context"
	"encoding/json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"testing"
)

func TestContextMessagesAfterTimelineAndBudget(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	session := s.Session()
	if err := s.AppendMessage(ctx, session.ID, modelapi.Message{Role: "user", Content: "already compacted"}); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordEvent(ctx, session.ID, run.Event{Kind: "compaction", Text: "summary committed"}); err != nil {
		t.Fatal(err)
	}
	records, _, err := s.Timeline(ctx, session.ID, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	through := records[len(records)-1].Sequence
	call := toolapi.Call{ID: "call", Name: "inspect_fixture", Arguments: json.RawMessage(`{"path":"README.md"}`)}
	for _, m := range []modelapi.Message{{Role: "user", Content: "new goal"}, {Role: "assistant", Calls: []toolapi.Call{call}}, {Role: "tool", ToolCallID: "call", ToolName: "inspect_fixture", Content: `{"content":"exact output"}`}} {
		if err := s.AppendMessage(ctx, session.ID, m); err != nil {
			t.Fatal(err)
		}
	}
	messages, err := s.ContextMessages(ctx, session.ID, through, 4096)
	if err != nil || len(messages) != 3 || messages[0].Content != "new goal" || messages[1].Calls[0].Name != "inspect_fixture" || messages[2].Content != `{"content":"exact output"}` {
		t.Fatalf("tail: %+v %v", messages, err)
	}
	all, err := s.ContextMessages(ctx, session.ID, 0, 4096)
	if err != nil || len(all) != 4 || all[0].Content != "already compacted" {
		t.Fatalf("all: %+v %v", all, err)
	}
	partial, err := s.ContextMessages(ctx, session.ID, 0, 1)
	if err == nil || len(partial) != 0 {
		t.Fatal("over-budget context silently returned incomplete history")
	}
}
