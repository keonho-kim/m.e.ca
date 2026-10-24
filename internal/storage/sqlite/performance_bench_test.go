// Purpose: Measure small history and usage projections in a session containing large original captures.
// Pattern: Synthetic storage benchmark.
// Usage: go test ./internal/storage/sqlite -run '^$' -bench BenchmarkCaptureProjections -benchmem.
// Related: internal/storage/sqlite/events.go, internal/storage/sqlite/timeline.go, internal/storage/sqlite/usage.go.
package sqlite

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
)

func BenchmarkCaptureProjections(b *testing.B) {
	ctx := context.Background()
	root := b.TempDir()
	sessions, err := OpenSessions(filepath.Join(root, "sessions"))
	if err != nil {
		b.Fatal(err)
	}
	s, err := sessions.Create(ctx, root, root)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { s.Close() })
	id := s.Session().ID
	capture, _ := json.Marshal(map[string]string{"Stdout": strings.Repeat("x", 1024*1024)})
	for range 40 {
		if err := s.RecordEvent(ctx, id, run.Event{Kind: "tool_succeeded", Text: "complete", Capture: capture, Output: json.RawMessage(`{"Stdout":"small excerpt"}`)}); err != nil {
			b.Fatal(err)
		}
	}
	if err := s.RecordEvent(ctx, id, run.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Input: 100, Output: 20}}); err != nil {
		b.Fatal(err)
	}
	b.Run("timeline", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, _, err := s.Timeline(ctx, id, 0, 100); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("usage", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := s.TokenUsage(ctx, id); err != nil {
				b.Fatal(err)
			}
		}
	})
	if err := s.Close(); err != nil {
		b.Fatal(err)
	}
	info, err := os.Stat(s.path)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(info.Size()), "database-bytes")
	b.Logf("40 x 1MiB captured originals: %d bytes after closing the DB", info.Size())
}
