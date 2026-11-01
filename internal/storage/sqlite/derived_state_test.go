// Purpose: Keep one compressed continuation and one monotonic occupancy sample per session.
// Pattern: Derived-state storage regression.
// Usage: Run TestDerivedStateReplacesSamplesAndMemories.
// Related: internal/storage/sqlite/memory.go, internal/storage/sqlite/schema.go.
package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
)

func TestDerivedStateReplacesSamplesAndMemories(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	previous := int64(0)
	for revision := 1; revision <= 20; revision++ {
		if err := s.AddContextTokens(ctx, id, compact.TokenCount{Amount: int64(revision), Kind: "request"}); err != nil {
			t.Fatal(err)
		}
		mark, err := s.MemoryWatermark(ctx, id, previous)
		if err != nil || mark.Usage <= previous || mark.Tokens != int64(revision) {
			t.Fatalf("sample lost monotonicity: %+v %v", mark, err)
		}
		previous = mark.Usage
		next := compact.Memory{Revision: revision, Through: int64(revision), UsageThrough: mark.Usage, Markdown: strings.Repeat("Requirements and observed results.\n", 2000)}
		if err := s.InstallMemory(ctx, id, revision-1, next); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"memories", "conversation_tokens"} {
		var count int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s retained %d records: %v", table, count, err)
		}
	}
	latest, err := s.Memory(ctx, id)
	if err != nil || latest.Revision != 20 || len(latest.Markdown) < 32768 {
		t.Fatalf("latest summary lost: %+v %v", latest, err)
	}
}
