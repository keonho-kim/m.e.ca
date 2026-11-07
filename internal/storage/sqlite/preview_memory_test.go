// Purpose: Bound preview allocation while preserving verification of the complete stored text.
// Pattern: Observable memory and integrity regression tests.
// Usage: go test ./internal/storage/sqlite -run TestPreview.
// Related: internal/storage/sqlite/timeline.go, internal/storage/content/text_prefix.go.
package sqlite

import (
	"runtime"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestPreviewDoesNotAllocateTheCompleteCompressedText(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.AppendMessage(t.Context(), s.Session().ID, model.Message{Role: "assistant", Content: strings.Repeat("x", 8<<20)}); err != nil {
		t.Fatal(err)
	}
	read := func() {
		t.Helper()
		page, _, err := s.TimelineSummaries(t.Context(), s.Session().ID, 0, 1, history.ReadBounds{MaxBytes: 32 << 20, TextBytes: 16 << 10})
		if err != nil || len(page) != 1 || !page[0].Truncated || len(page[0].Message.Content) != 16<<10 {
			t.Fatal("preview changed", err)
		}
	}
	read() // Initialize database, JSON and decoder resources outside the observation.
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	read()
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 2<<20 {
		t.Fatalf("16KiB preview allocated %d bytes for an 8MiB compressed string", allocated)
	}
}

func TestPreviewRejectsCorruptionBeyondItsVisiblePrefix(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.AppendMessage(t.Context(), s.Session().ID, model.Message{Role: "assistant", Content: strings.Repeat("x", 128<<10)}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(t.Context(), "UPDATE payloads SET digest=zeroblob(32)"); err != nil {
		t.Fatal(err)
	}

	if _, _, err := s.TimelineSummaries(t.Context(), s.Session().ID, 0, 1, history.ReadBounds{MaxBytes: 1 << 20, TextBytes: 16}); err == nil {
		t.Fatal("preview returned text before checking the entire stored payload digest")
	}
}

func TestPreviewSharesRepeatedContentWithinItsPageBudget(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for range 2 {
		if err := s.AppendMessage(t.Context(), s.Session().ID, model.Message{Role: "assistant", Content: strings.Repeat("x", 1<<20)}); err != nil {
			t.Fatal(err)
		}
	}
	page, _, err := s.TimelineSummaries(t.Context(), s.Session().ID, 0, 2, history.ReadBounds{MaxBytes: (1 << 20) + 1024, TextBytes: 16384})
	if err != nil || len(page) != 2 {
		t.Fatal("repeated preview exhausted the page budget", err)
	}
}
