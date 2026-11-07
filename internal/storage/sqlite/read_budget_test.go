// Purpose: Enforce restoration bounds before payload hydration and preserve complete stored originals.
// Pattern: SQLite failure-path and preview contract tests.
// Usage: go test ./internal/storage/sqlite -run TestReadBudget -bench BenchmarkServerPreview.
// Related: internal/storage/sqlite/read_budget.go, internal/storage/sqlite/timeline.go.
package sqlite

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestReadBudgetRejectsOriginalWithoutChangingIt(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id := s.Session().ID
	text := strings.Repeat("x", 2<<20)
	if err := s.AppendMessage(t.Context(), id, model.Message{Role: "assistant", Content: text}); err != nil {
		t.Fatal(err)
	}
	through, err := s.OutputPosition(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResponseOutput(t.Context(), id, 0, through, 1<<20); !errors.Is(err, history.ErrReadLimit) {
		t.Fatal("large payload reached restoration", err)
	}
	page, _, err := s.TimelineSummaries(t.Context(), id, 0, 1, history.ReadBounds{MaxBytes: 4 << 20, TextBytes: 16 << 10})
	if err != nil || len(page) != 1 || len(page[0].Message.Content) != 16<<10 || !page[0].Truncated {
		t.Fatal("preview not bounded", err)
	}
	got, err := s.ResponseOutput(t.Context(), id, 0, through, 0)
	if err != nil || got != text {
		t.Fatal("budget rejection changed original", err)
	}
}

func TestReadBudgetExcludesPrivateContinuation(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id := s.Session().ID
	if err := s.AppendMessage(t.Context(), id, model.Message{Role: "assistant", Content: "public", Reasoning: strings.Repeat("private", 100000)}); err != nil {
		t.Fatal(err)
	}
	page, _, err := s.TimelineSummaries(t.Context(), id, 0, 1, history.ReadBounds{MaxBytes: 64 << 10, TextBytes: 16 << 10})
	if err != nil || len(page) != 1 || page[0].Message.Content != "public" || page[0].Message.Reasoning != "" {
		t.Fatal("private payload entered preview", err)
	}
}

func TestDisplayMetadataPageFitsReadBudget(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for range 4 {
		if err := s.AppendMessage(t.Context(), s.Session().ID, model.Message{Role: "assistant", Content: strings.Repeat("x", 1024)}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.displayPage(t.Context(), s.Session().ID, 1<<62, 100, 1600)
	if err != nil || len(page) == 0 {
		t.Fatal(err)
	}
	bytes := 0
	for _, entry := range page {
		bytes += len(entry.record.body) + len(entry.record.refs) + len(entry.arguments)
	}
	if bytes > 1600 {
		t.Fatal("metadata page exceeded restoration budget before hydration", bytes)
	}
}

func BenchmarkServerPreview8MiB(b *testing.B) {
	catalog, err := OpenSessions(filepath.Join(b.TempDir(), "sessions"))
	if err != nil {
		b.Fatal(err)
	}
	project := b.TempDir()
	s, err := catalog.Create(b.Context(), project, project)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	if err := s.AppendMessage(b.Context(), s.Session().ID, model.Message{Role: "assistant", Content: strings.Repeat("x", 8<<20)}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		page, _, err := s.TimelineSummaries(b.Context(), s.Session().ID, 0, 1, history.ReadBounds{MaxBytes: 32 << 20, TextBytes: 16 << 10})
		if err != nil || len(page) != 1 || !page[0].Truncated {
			b.Fatal(err)
		}
	}
}
