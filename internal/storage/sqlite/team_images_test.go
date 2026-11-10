// Purpose: Keep public user images durable while private team journals store only authenticated references.
// Pattern: SQLite ownership and restore integration tests.
// Usage: go test ./internal/storage/sqlite -run TestTeamUserImages.
// Related: internal/storage/sqlite/team_images.go, internal/storage/sqlite/image_boundary.go.
package sqlite

import (
	"bytes"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestTeamUserImagesUsePublicOwnerAndPinUnreadInput(t *testing.T) {
	root, child := teamStorage(t)
	image, err := media.New("user.png", "image/png", "png", 128, 128, bytes.Repeat([]byte("original-image-pixels"), 4096))
	if err != nil {
		t.Fatal(err)
	}
	public, err := root.AppendTeamMessage(t.Context(), channel.Message{From: "user", Images: []media.Image{image}})
	if err != nil {
		t.Fatal(err)
	}
	message := public.Context()
	if err := child.CommitContext(t.Context(), child.session.ID, []record.Entry{{Message: &message}}, public.Sequence); err != nil {
		t.Fatal(err)
	}
	var storedBytes int
	var references int
	if err := child.db.QueryRow("SELECT count(*) FROM payloads WHERE codec=4").Scan(&references); err != nil || references != 1 {
		t.Fatal("missing public image reference", references, err)
	}
	if err := child.db.QueryRow("SELECT sum(length(body)) FROM payloads").Scan(&storedBytes); err != nil {
		t.Fatal(err)
	}
	if storedBytes > 4096 {
		t.Fatalf("private journal duplicated %d image bytes", storedBytes)
	}
	values, err := child.ContextMessages(t.Context(), child.session.ID, 0, 2<<20, true, nil)
	if err != nil || len(values) != 1 || len(values[0].Images) != 1 || !bytes.Equal(values[0].Images[0].Data, image.Data) {
		t.Fatalf("image reference failed: %v", err)
	}
	if _, err := child.ContextMessages(t.Context(), child.session.ID, 0, 8192, false, nil); err != nil {
		t.Fatal("text model loaded image reference", err)
	}
	mark, err := child.MemoryWatermark(t.Context(), child.session.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := child.CompactionBoundary(t.Context(), child.session.ID, mark.Timeline, 0)
	if err != nil || boundary != 0 {
		t.Fatal("unread user image compacted", boundary, err)
	}
}
