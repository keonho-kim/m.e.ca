// Purpose: Measure phase-local allocation and history-scan costs independently of SQLite I/O.
// Pattern: Fixed-workload microbenchmarks.
// Usage: Run BenchmarkJournal and BenchmarkPrivateWatermark with -benchmem.
// Related: internal/agent/lat/transcript/journal.go, internal/agent/lat/transcript/compaction.go.
package transcript

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func BenchmarkJournalImageCommit(b *testing.B) {
	image, err := media.New("image.png", "image/png", "png", 1024, 768, make([]byte, 1<<20))
	if err != nil {
		b.Fatal(err)
	}
	j, err := New(&failingCommitter{}, "session", "parent_run", "review")
	if err != nil {
		b.Fatal(err)
	}
	defer j.Close()
	message := modelapi.Message{Role: "tool", ToolCallID: "image_call", ToolName: "read_image", Images: []media.Image{image}}
	entries := []runrecord.Entry{{Message: &message}}
	b.ReportAllocs()
	for b.Loop() {
		if err := j.Commit(context.Background(), "session", entries); err != nil {
			b.Fatal(err)
		}
		clear(j.messages)
		j.messages = j.messages[:0]
	}
}

func BenchmarkPrivateWatermark(b *testing.B) {
	for _, count := range []int{400, 4000} {
		b.Run(fmt.Sprintf("messages%d", count), func(b *testing.B) {
			j, err := New(&failingCommitter{}, "session", "parent_run", "review")
			if err != nil {
				b.Fatal(err)
			}
			defer j.Close()
			for range count {
				message := modelapi.Message{Role: "user", Content: strings.Repeat("a", 100)}
				if err := j.Commit(context.Background(), "session", []runrecord.Entry{{Message: &message}}); err != nil {
					b.Fatal(err)
				}
			}
			if err := j.AddContextTokens(context.Background(), "session", compact.TokenCount{Amount: 100, Kind: "request"}); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := j.MemoryWatermark(context.Background(), "session", 0); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
