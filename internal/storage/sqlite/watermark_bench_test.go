// Purpose: Measure current-context lookups without hydrating conversation bodies or adding stored state.
// Pattern: Populated-session SQLite benchmark.
// Usage: go test ./internal/storage/sqlite -run '^$' -bench BenchmarkMemoryWatermark -benchmem.
// Related: internal/storage/sqlite/memory.go, internal/storage/sqlite/context_usage_test.go.
package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func BenchmarkMemoryWatermark(b *testing.B) {
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
	for i := range 1000 {
		if err := s.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: strings.Repeat("x", 300)}); err != nil {
			b.Fatal(err)
		}
		if i == 949 {
			if err := s.AddContextTokens(ctx, id, compact.TokenCount{Amount: 100000, Kind: "request"}); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		mark, err := s.MemoryWatermark(ctx, id, 0)
		if err != nil || mark.Tokens != 105000 || !mark.Estimated {
			b.Fatalf("watermark=%+v error=%v", mark, err)
		}
	}
}
