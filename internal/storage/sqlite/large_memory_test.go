// Purpose: Verify expanded compaction summaries survive SQLite storage and restoration.
// Pattern: Repository boundary regression test.
// Usage: go test ./internal/storage/sqlite -run TestLargeMemory.
// Related: internal/storage/sqlite/memory.go, internal/agent/compact/usage.go.
package sqlite

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/compact"
	"strings"
	"testing"
)

func TestLargeMemoryRoundTrip(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	memo := compact.Memory{Revision: 1, Markdown: strings.Repeat("source <text>\n", 10000)}
	if err := s.InstallMemory(context.Background(), s.Session().ID, 0, memo); err != nil {
		t.Fatal(err)
	}
	got, err := s.Memory(context.Background(), s.Session().ID)
	if err != nil || got != memo {
		t.Fatalf("large summary changed: %v", err)
	}
}
