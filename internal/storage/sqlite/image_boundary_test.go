// Purpose: Keep unread image groups across interruption and respect the requested timeline cutoff.
// Pattern: Compaction boundary regression tests.
// Usage: SQLite compaction tests distinguish completed generations from interrupted drafts and events.
// Related: internal/storage/sqlite/image_boundary.go, internal/storage/sqlite/memory_boundary.go.
package sqlite

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestUnreadImageBoundaryUsesCompletedGenerations(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx, id := context.Background(), s.Session().ID
	picture, _ := media.New("shot.png", "image/png", "png", 8, 8, []byte("snapshot"))
	call := toolapi.Call{ID: "image", Name: "read_image", Arguments: json.RawMessage(`{"path":"shot.png"}`)}
	for _, message := range []modelapi.Message{
		{Role: "user", Content: "inspect"},
		{Role: "assistant", Content: "earlier response"},
		{Role: "assistant", Calls: []toolapi.Call{call}},
		{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Images: []media.Image{picture}},
		{Role: "assistant", Content: "unfinished observation", Incomplete: true},
	} {
		if err := s.AppendMessage(ctx, id, message); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RecordEvent(ctx, id, runrecord.Event{Kind: "thinking", Thinking: "still inspecting"}); err != nil {
		t.Fatal(err)
	}
	mark, err := s.MemoryWatermark(ctx, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	unreadThrough := mark.Timeline
	for _, keep := range []int{0, 1, 8192} {
		boundary, err := s.CompactionBoundary(ctx, id, unreadThrough, keep)
		if err != nil || boundary > 2 {
			t.Fatalf("unread image group compacted: %d %v", boundary, err)
		}
	}
	if err := s.AppendMessage(ctx, id, modelapi.Message{Role: "assistant", Content: "observed"}); err != nil {
		t.Fatal(err)
	}
	boundary, err := s.CompactionBoundary(ctx, id, unreadThrough, 0)
	if err != nil || boundary != 2 {
		t.Fatalf("response outside cutoff consumed an image: %d %v", boundary, err)
	}
	boundary, err = s.CompactionBoundary(ctx, id, unreadThrough+1, 0)
	if err != nil || boundary != unreadThrough+1 {
		t.Fatalf("completed response did not release the image: %d %v", boundary, err)
	}
}
