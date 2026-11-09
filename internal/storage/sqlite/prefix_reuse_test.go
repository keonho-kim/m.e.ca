// Purpose: Verify reuse of shared JSON text while paging bounded history previews.
// Pattern: SQLite integration regression test.
// Usage: go test ./internal/storage/sqlite.
// Related: internal/storage/sqlite/page_references.go, internal/storage/sqlite/page_prefixes.go, internal/storage/content/reference.go.
package sqlite

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestSharedJSONTextPreviewChargedOnce(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	text := `{"items":"` + strings.Repeat("x", 40000) + `"}`
	for range 4 {
		if err := store.AppendMessage(t.Context(), store.Session().ID, model.Message{Role: "assistant", Content: text}); err != nil {
			t.Fatal(err)
		}
	}
	page, _, err := store.TimelineSummaries(t.Context(), store.Session().ID, 0, 4, history.ReadBounds{MaxBytes: 64 << 10, TextBytes: 128})
	if err != nil || len(page) != 4 {
		t.Fatalf("shared %d-byte original did not yield four 128-byte previews within 64 KiB: count=%d err=%v", len(text), len(page), err)
	}
}

func TestSharedJSONTextPreviewFitsDefaultServerBudget(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	text := `{"items":"` + strings.Repeat("x", 20<<20) + `"}`
	for range 2 {
		if err := store.AppendMessage(t.Context(), store.Session().ID, model.Message{Role: "assistant", Content: text}); err != nil {
			t.Fatal(err)
		}
	}
	page, _, err := store.TimelineSummaries(t.Context(), store.Session().ID, 0, 2, history.ReadBounds{MaxBytes: 32 << 20, TextBytes: 16 << 10})
	if err != nil || len(page) != 2 {
		t.Fatalf("two previews of one shared %d-byte original exceeded default server budget: count=%d err=%v", len(text), len(page), err)
	}
}

func TestSharedPayloadTextViewsStayDistinct(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	plain := strings.Repeat("x", 40000)
	quoted := `"` + plain + `"`
	for _, text := range []string{plain, quoted} {
		if err := store.AppendMessage(t.Context(), store.Session().ID, model.Message{Role: "assistant", Content: text}); err != nil {
			t.Fatal(err)
		}
	}
	page, _, err := store.TimelineSummaries(t.Context(), store.Session().ID, 0, 2, history.ReadBounds{MaxBytes: 128 << 10, TextBytes: 128})
	if err != nil || len(page) != 2 || page[0].Message.Content != plain[:128] || page[1].Message.Content != quoted[:128] {
		t.Fatal("JSON value and text view shared a decoded prefix", page, err)
	}
}
