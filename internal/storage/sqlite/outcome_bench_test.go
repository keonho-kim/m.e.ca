// Purpose: Compare split and atomic outcome persistence with identical records and final storage sizes.
// Pattern: Fixed-payload SQLite durability benchmark.
// Usage: go test ./internal/storage/sqlite -run '^$' -bench BenchmarkOutcomeCommit -benchtime=100x -benchmem.
// Related: internal/storage/sqlite/commit.go, internal/agent/run/execution_result.go.
package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
)

func BenchmarkOutcomeCommit(b *testing.B) {
	for _, atomic := range []bool{false, true} {
		name := "split"
		if atomic {
			name = "atomic"
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
			message := modelapi.Message{Role: "tool", ToolCallID: "call", ToolName: "write_file", Content: `{"changed":true}`}
			outcome := run.Event{Kind: "tool_succeeded", Text: "created"}
			checkpoint := run.Event{Kind: "thinking", Thinking: "checkpoint payload"}
			records := []run.JournalRecord{{Event: &outcome}, {Message: &message}, {Event: &checkpoint}}
			b.ReportAllocs()
			for b.Loop() {
				if atomic {
					err = s.Commit(ctx, id, records)
				} else {
					err = s.Commit(ctx, id, records[:1])
					if err == nil {
						err = s.Commit(ctx, id, records[1:])
					}
				}
				if err != nil {
					b.Fatal(err)
				}
			}
			if err := s.Close(); err != nil {
				b.Fatal(err)
			}
			info, err := os.Stat(s.path)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(info.Size()), "database-bytes")
		})
	}
}
