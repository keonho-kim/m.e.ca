// Purpose: Reproduce visibility of an image-only user message.
// Pattern: Storage visibility regression tests.
// Usage: go test ./internal/storage/sqlite -run TestRegression.
// Related: internal/storage/sqlite/display_page.go.
package sqlite

import (
	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/storage/history"
	"strings"
	"testing"
)

func TestRegressionImageOnlyVisibleInHistory(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	picture, err := media.New("shot.png", "image/png", "png", 8, 8, []byte("snapshot"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AppendMessage(t.Context(), s.Session().ID, modelapi.Message{Role: "user", Images: []media.Image{picture}}); err != nil {
		t.Fatal(err)
	}
	rows, _, err := s.TimelineSummaries(t.Context(), s.Session().ID, 0, 32, history.ReadBounds{MaxBytes: 32 << 20, TextBytes: 16 << 10})
	if err != nil || len(rows) != 1 {
		t.Fatalf("stored image-only message history rows=%d error=%v", len(rows), err)
	}
}

func TestRegressionLargeMessageHistoryPreview(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.AppendMessage(t.Context(), s.Session().ID, modelapi.Message{Role: "assistant", Content: strings.Repeat("a", 33<<20)}); err != nil {
		t.Fatal(err)
	}
	rows, _, err := s.TimelineSummaries(t.Context(), s.Session().ID, 0, 1, history.ReadBounds{MaxBytes: 32 << 20, TextBytes: 16 << 10})
	if err != nil || len(rows) != 1 {
		t.Fatalf("16 KiB history preview of valid 33 MiB message: rows=%d error=%v", len(rows), err)
	}
}
