// Purpose: Preserve inline and external image reads with admission and integrity checks.
// Pattern: Isolated SQLite repository contract tests.
// Usage: go test ./internal/storage/sqlite -run TestHistoryImage.
// Related: internal/storage/sqlite/history_image.go, internal/storage/content/pack.go.
package sqlite

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestHistoryImageReadsInlineAndReferencedBytes(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	// These encoded digits form valid JSON, so Pack writes a string reference.
	quoted, err := base64.StdEncoding.DecodeString(strings.Repeat("1234", 10000))
	if err != nil {
		t.Fatal(err)
	}
	payloads := [][]byte{[]byte("inline bytes"), bytes.Repeat([]byte("pixels"), 10000), quoted}
	images := make([]media.Image, 0, len(payloads))
	for _, body := range payloads {
		image, err := media.New("shot.png", "image/png", "png", 4, 4, body)
		if err != nil {
			t.Fatal(err)
		}
		images = append(images, image)
	}
	if err := store.AppendMessage(t.Context(), store.Session().ID, modelapi.Message{Role: "user", Images: images}); err != nil {
		t.Fatal(err)
	}
	for index, want := range images {
		actual, err := store.HistoryImage(t.Context(), 1, index, history.ReadBounds{MaxBytes: 1 << 20})
		if err != nil || actual.Digest != want.Digest || !bytes.Equal(actual.Data, want.Data) {
			t.Fatalf("image %d: %v", index, err)
		}
	}
	if _, err := store.HistoryImage(t.Context(), 1, 1, history.ReadBounds{MaxBytes: 1024}); !errors.Is(err, history.ErrReadLimit) {
		t.Fatalf("admission: %v", err)
	}
	if _, err := store.db.ExecContext(t.Context(), "UPDATE messages SET body=json_set(body,'$.images[1].data','YWJj')"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.HistoryImage(t.Context(), 1, 1, history.ReadBounds{}); err == nil {
		t.Fatal("accepted a referenced image without its null placeholder")
	}
}
