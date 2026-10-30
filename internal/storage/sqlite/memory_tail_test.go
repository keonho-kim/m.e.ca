// Purpose: Compact older large native results while retaining recent pairs and original records.
// Pattern: Session-storage integration test.
// Usage: go test ./internal/storage/sqlite -run TestMemoryTailSkipsOlderLargeResult.
// Related: internal/storage/history/boundary.go, internal/storage/sqlite/memory_boundary.go.
package sqlite

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestMemoryTailSkipsOlderLargeResult(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, id := context.Background(), store.Session().ID
	for _, pair := range []struct{ id, output string }{{"old", strings.Repeat("large listing\n", 3000)}, {"recent", "current observation"}} {
		call := toolapi.Call{ID: pair.id, Name: "execute_command", Arguments: json.RawMessage(`{"command":"cat source"}`)}
		for _, message := range []modelapi.Message{{Role: "assistant", Calls: []toolapi.Call{call}}, {Role: "tool", ToolName: call.Name, ToolCallID: call.ID, Content: pair.output}} {
			if err := store.AppendMessage(ctx, id, message); err != nil {
				t.Fatal(err)
			}
		}
	}
	mark, err := store.MemoryWatermark(ctx, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := store.CompactionBoundary(ctx, id, mark.Timeline, 1024)
	if err != nil || boundary != 2 {
		t.Fatalf("boundary=%d err=%v", boundary, err)
	}
	if err := store.InstallMemory(ctx, id, 0, compact.Memory{Revision: 1, Through: boundary, Markdown: "- Earlier listing observed"}); err != nil {
		t.Fatal(err)
	}
	tail, err := store.ContextMessages(ctx, id, boundary, 1024, true)
	if err != nil || len(tail) != 2 || tail[0].Calls[0].ID != "recent" || tail[1].ToolCallID != "recent" {
		t.Fatalf("tail=%+v err=%v", tail, err)
	}
	original, err := store.ContextMessages(ctx, id, 0, 16*1024*1024, true)
	if err != nil || len(original) != 4 || !strings.Contains(original[1].Content, "large listing") {
		t.Fatalf("original history lost: count=%d err=%v", len(original), err)
	}
}
