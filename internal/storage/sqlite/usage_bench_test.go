// Purpose: Measure verified warm usage reads as persisted usage histories grow.
// Pattern: Storage scaling benchmark.
// Usage: go test ./internal/storage/sqlite -run '^$' -bench BenchmarkTokenUsage -benchmem.
// Related: internal/storage/sqlite/usage.go, internal/storage/sqlite/usage_incremental_test.go.
package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func BenchmarkTokenUsage(b *testing.B) {
	for _, count := range []int{100, 1000, 10000, 50000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			ctx := context.Background()
			project := b.TempDir()
			sessions, err := OpenSessions(filepath.Join(b.TempDir(), "sessions"))
			if err != nil {
				b.Fatal(err)
			}
			store, err := sessions.Create(ctx, project, project)
			if err != nil {
				b.Fatal(err)
			}
			defer store.Close()
			_, err = store.db.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<?)
			 INSERT INTO tool_events(kind,body,raw_bytes,created_at)
			 SELECT 'usage','{"kind":"usage","usage":{"input":13,"output":7}}',48,'2026-11-11T00:00:00Z' FROM n`, count)
			if err != nil {
				b.Fatal(err)
			}
			id := store.Session().ID
			if _, err := store.TokenUsage(ctx, id); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				total, err := store.TokenUsage(ctx, id)
				if err != nil || total.Input != int64(count)*13 || total.Output != int64(count)*7 {
					b.Fatalf("incorrect usage: %+v %v", total, err)
				}
			}
		})
	}
}
