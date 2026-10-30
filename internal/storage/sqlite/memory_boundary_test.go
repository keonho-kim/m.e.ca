// Purpose: Verify compaction retains recent original messages without splitting their contents.
// Pattern: Repository behavior test.
// Usage: go test ./internal/storage/sqlite -run TestCompactionTail.
// Related: internal/storage/sqlite/memory_boundary.go, internal/storage/sqlite/context.go.
package sqlite

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"testing"
)

func TestCompactionTail(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	}()
	ctx := context.Background()
	session := store.Session()
	for _, text := range []string{"older", "recent exact content"} {
		if err := store.AppendMessage(ctx, session.ID, modelapi.Message{Role: "user", Content: text}); err != nil {
			t.Fatal(err)
		}
	}
	mark, err := store.MemoryWatermark(ctx, session.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := store.CompactionBoundary(ctx, session.ID, mark.Timeline, 1)
	if err != nil {
		t.Fatal(err)
	}
	messages, err := store.ContextMessages(ctx, session.ID, boundary, 8192, true)
	if err != nil || len(messages) != 1 || messages[0].Content != "recent exact content" {
		t.Fatalf("tail=%+v err=%v", messages, err)
	}
	boundary, err = store.CompactionBoundary(ctx, session.ID, mark.Timeline, 8192)
	if err != nil || boundary != 0 {
		t.Fatalf("small conversation was compacted: %d %v", boundary, err)
	}
}
