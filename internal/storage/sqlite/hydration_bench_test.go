// Purpose: Measure page restoration with repeated and distinct immutable payloads.
// Pattern: Fixed-history storage benchmark.
// Usage: Run BenchmarkPayloadPages with -benchmem for export and display paths.
// Related: internal/storage/sqlite/export.go, internal/storage/sqlite/timeline.go.
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func BenchmarkPayloadPages(b *testing.B) {
	for _, shared := range []bool{false, true} {
		b.Run(fmt.Sprintf("shared%t", shared), func(b *testing.B) {
			s := benchmarkLATStore(b)
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

func BenchmarkLATOutcomePages(b *testing.B) {
	for _, shared := range []bool{false, true} {
		b.Run(fmt.Sprintf("shared%t", shared), func(b *testing.B) {
			s := benchmarkLATStore(b)
			ctx, id := context.Background(), s.Session().ID
			for i := range 32 {
				text := strings.Repeat("Actual command output.\n", 8192)
				if !shared {
					text += fmt.Sprint(i)
				}
				output, err := json.Marshal(map[string]string{"stdout": text})
				if err != nil {
					b.Fatal(err)
				}
				if err := s.RecordEvent(ctx, id, runrecord.Event{Kind: "tool_succeeded", Text: "Observed result", Output: output, Operation: &toolapi.Operation{Name: "execute_command", Status: "tool_succeeded", Result: "Observed result"}}); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			for b.Loop() {
				if records, _, err := s.LATRecords(ctx, id, 0, 1<<62, latmemory.MaxBytes); err != nil || len(records) != 32 {
					b.Fatalf("outcomes=%d: %v", len(records), err)
				}
			}
		})
	}
}
