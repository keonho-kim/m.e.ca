// Purpose: Measure initial conversation discovery and grouped journal commit costs.
// Pattern: Synthetic storage benchmarks.
// Usage: go test ./internal/storage/sqlite -run '^$' -bench 'Benchmark(Conversations|Journal)' -benchmem.
// Related: internal/storage/sqlite/conversations.go, internal/storage/sqlite/commit.go.
package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
)

func BenchmarkConversations(b *testing.B) {
	root := b.TempDir()
	catalog, err := OpenSessions(filepath.Join(root, "sessions"))
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	for range 200 {
		s, err := catalog.Create(ctx, root, root)
		if err != nil {
			b.Fatal(err)
		}
		if err := s.Close(); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := catalog.Conversations(ctx, ""); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJournal(b *testing.B) {
	for _, grouped := range []bool{false, true} {
		name := "single"
		if grouped {
			name = "grouped"
		}
		b.Run(name, func(b *testing.B) {
			root := b.TempDir()
			catalog, err := OpenSessions(filepath.Join(root, "sessions"))
			if err != nil {
				b.Fatal(err)
			}
			ctx := context.Background()
			s, err := catalog.Create(ctx, root, root)
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { s.Close() })
			id := s.Session().ID
			m := modelapi.Message{Role: "user", Content: "request"}
			e := run.Event{Kind: "thinking", Thinking: "observed"}
			records := []run.JournalRecord{{Message: &m}, {Event: &e}, {Event: &e}}
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				if grouped {
					err = s.Commit(ctx, id, records)
				} else {
					for _, r := range records {
						if err = s.Commit(ctx, id, []run.JournalRecord{r}); err != nil {
							break
						}
					}
				}
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
