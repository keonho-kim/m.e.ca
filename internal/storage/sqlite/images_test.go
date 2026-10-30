// Purpose: Preserve image snapshots through content storage while keeping terminal and summary projections small.
// Pattern: Session repository integration tests.
// Usage: go test ./internal/storage/sqlite -run TestImage.
// Related: internal/storage/sqlite/image_boundary.go, internal/storage/sqlite/timeline.go.
package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestImagePersistenceCompactionAndDisplay(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, id := context.Background(), store.Session().ID
	image, err := media.New("shot.png", "image/png", "png", 16, 16, bytes.Repeat([]byte("compressed-original"), 4096))
	if err != nil {
		t.Fatal(err)
	}
	invocation := toolapi.Call{ID: "read", Name: "read_image", Arguments: json.RawMessage(`{"path":"shot.png"}`)}
	for _, message := range []modelapi.Message{{Role: "user", Content: "Inspect this image"}, {Role: "assistant", Calls: []toolapi.Call{invocation}}, {Role: "tool", ToolName: "read_image", ToolCallID: "read", Content: `{"Path":"shot.png","Width":16,"Height":16}`, Images: []media.Image{image}}, {Role: "user", Content: "Also check the spacing"}} {
		if err := store.AppendMessage(ctx, id, message); err != nil {
			t.Fatal(err)
		}
	}
	mark, err := store.MemoryWatermark(ctx, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, keep := range []int{0, 1, 8192} {
		boundary, err := store.CompactionBoundary(ctx, id, mark.Timeline, keep)
		if err != nil || boundary > 1 {
			t.Fatalf("unread image compacted: boundary=%d keep=%d err=%v", boundary, keep, err)
		}
	}
	restored, err := store.ContextMessages(ctx, id, 0, 16*1024*1024, true)
	if err != nil || len(restored) != 4 || len(restored[2].Images) != 1 || !bytes.Equal(restored[2].Images[0].Data, image.Data) {
		t.Fatalf("snapshot lost: count=%d err=%v", len(restored), err)
	}
	timeline, _, err := store.Timeline(ctx, id, 0, 10)
	if err != nil || len(timeline[2].Message.Images) != 0 {
		t.Fatal("terminal history hydrates image binary", err)
	}
	records, _, err := store.MemoryRecords(ctx, id, 0, mark.Timeline)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(records)
	if bytes.Contains(body, []byte("Y29tcHJlc3NlZC")) {
		t.Fatal("image binary leaked to summary input")
	}
	// Once a completed generation follows the image, the old group may be compacted.
	if err := store.AppendMessage(ctx, id, modelapi.Message{Role: "assistant", Content: "Observed layout"}); err != nil {
		t.Fatal(err)
	}
	mark, err = store.MemoryWatermark(ctx, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := store.CompactionBoundary(ctx, id, mark.Timeline, 0)
	if err != nil || boundary != mark.Timeline {
		t.Fatalf("consumed image remained pinned: %d %v", boundary, err)
	}
}

func TestImageTextContextSkipsPixelsAndTheirBudget(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx, id := context.Background(), s.Session().ID
	picture, _ := media.New("shot.png", "image/png", "png", 32, 32, bytes.Repeat([]byte{42}, 1<<20))
	if err := s.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: "Inspect", Images: []media.Image{picture}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("UPDATE payloads SET body=x'00'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ContextMessages(ctx, id, 0, 16<<20, true); err == nil {
		t.Fatal("fixture did not make the original image unavailable")
	}
	got, err := s.ContextMessages(ctx, id, 0, 8192, false)
	if err != nil || len(got) != 1 || len(got[0].Images) != 0 || !strings.Contains(got[0].Content, modelapi.ImageOmissionNotice) {
		t.Fatalf("text context restored omitted pixels or counted their size: %v", err)
	}
}
