// Purpose: Measure LAT history hydration and scan cursors against equivalent public records.
// Pattern: Fixed-input persistence benchmark.
// Usage: Run BenchmarkLATTimeline and BenchmarkLATScanCursor with -benchmem.
// Related: internal/storage/sqlite/timeline.go, internal/storage/sqlite/lat_sources.go.
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func benchmarkLATStore(b *testing.B) *Store {
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

func BenchmarkLATTimeline(b *testing.B) {
	for _, internal := range []bool{false, true} {
		b.Run(fmt.Sprint(internal), func(b *testing.B) {
			s := benchmarkLATStore(b)
			ctx, id := context.Background(), s.Session().ID
			output, _ := json.Marshal(map[string]any{"Stdout": strings.Repeat("Observed output line.\n", 80), "ExitCode": 0})
			entries := make([]runrecord.Entry, 100)
			for i := range entries {
				call := toolapi.Call{ID: fmt.Sprintf("call_%d", i), Name: "execute_command", Arguments: json.RawMessage(`{"command":"cat README.md","workdir":"."}`)}
				e := runrecord.Event{Kind: "tool_succeeded", Call: &call, CallID: call.ID, Output: output}
				if internal {
					e.Internal = true
					e.LAT = &runrecord.LATRun{ID: "lat_review", ParentRunID: "run_main", Phase: "review"}
				}
				entries[i].Event = &e
			}
			if err := s.Commit(ctx, id, entries); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, _, err := s.Timeline(ctx, id, 0, 100); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkLATScanCursor(b *testing.B) {
	for _, count := range []int{0, 1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			s := benchmarkLATStore(b)
			ctx, id := context.Background(), s.Session().ID
			if err := s.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: "Inspect sources"}); err != nil {
				b.Fatal(err)
			}
			entries := make([]runrecord.Entry, count)
			for i := range entries {
				entries[i].Event = &runrecord.Event{Kind: "thinking", Thinking: "Internal observation", Internal: true, LAT: &runrecord.LATRun{ID: "lat_plan", ParentRunID: "run_main", Phase: "plan"}}
			}
			if count > 0 {
				if err := s.Commit(ctx, id, entries); err != nil {
					b.Fatal(err)
				}
			}
			mark, err := s.LATWatermark(ctx, id, 0)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := s.LATWatermark(ctx, id, mark.Scanned); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
