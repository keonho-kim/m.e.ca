// Purpose: Measure page restoration with repeated and distinct immutable payloads.
// Pattern: Fixed-history storage benchmark.
// Usage: Run BenchmarkPayloadPages with -benchmem for export and display paths.
// Related: internal/storage/sqlite/export.go, internal/storage/sqlite/timeline.go.
package sqlite

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func BenchmarkPayloadPages(b *testing.B) {
	for _, shared := range []bool{false, true} {
		b.Run(fmt.Sprintf("shared%t", shared), func(b *testing.B) {
			s := benchmarkPayloadStore(b)
			ctx, id := context.Background(), s.Session().ID
			for i := range 32 {
				text := strings.Repeat("Observed original reasoning line.\n", 4096)
				if !shared {
					text += fmt.Sprint(i)
				}
				if err := s.RecordEvent(ctx, id, runrecord.Event{Kind: "thinking", Thinking: text}); err != nil {
					b.Fatal(err)
				}
			}
			b.Run("export", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if err := s.Export(ctx, id, io.Discard); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("timeline", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, _, err := s.Timeline(ctx, id, 0, 64); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func benchmarkPayloadStore(b *testing.B) *Store {
	b.Helper()
	root := b.TempDir()
	catalog, err := OpenSessions(filepath.Join(root, "sessions"))
	if err != nil {
		b.Fatal(err)
	}
	s, err := catalog.Create(context.Background(), root, root)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { s.Close() })
	return s
}
