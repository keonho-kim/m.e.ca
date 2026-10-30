// Purpose: Measure small image-history projections and compaction boundaries as original history grows.
// Pattern: Synthetic session workload benchmarks.
// Usage: Run BenchmarkImageHistory and BenchmarkImageBoundary with -benchmem.
// Related: internal/storage/sqlite/input_history.go, internal/storage/sqlite/image_boundary.go.
package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func imageBenchmarkStore(b *testing.B) *Store {
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

func BenchmarkImageHistory(b *testing.B) {
	s := imageBenchmarkStore(b)
	ctx, id := context.Background(), s.Session().ID
	data := make([]byte, 1<<20)
	var seed uint32 = 42
	for i := range data {
		seed ^= seed << 13
		seed ^= seed >> 17
		seed ^= seed << 5
		data[i] = byte(seed)
	}
	picture, err := media.New("shot.png", "image/png", "png", 1024, 768, data)
	if err != nil {
		b.Fatal(err)
	}
	messages := []modelapi.Message{{Role: "user", Content: "inspect"}}
	for i := range 10 {
		messages = append(messages, modelapi.Message{Role: "tool", ToolName: "read_image", ToolCallID: fmt.Sprintf("read%d", i), Images: []media.Image{picture}})
	}
	messages = append(messages, modelapi.Message{Role: "assistant", Content: "completed"})
	entries := make([]runrecord.Entry, len(messages))
	for i := range messages {
		entries[i].Message = &messages[i]
	}
	if err := s.Commit(ctx, id, entries); err != nil {
		b.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		read func() error
	}{
		{"original_context", func() error { _, err := s.ContextMessages(ctx, id, 0, 16<<20, true); return err }},
		{"text_context", func() error { _, err := s.ContextMessages(ctx, id, 0, 8192, false); return err }},
		{"recent_inputs", func() error { _, err := s.RecentInputs(ctx, id, 200); return err }},
		{"pending_tools", func() error { _, err := s.PendingTools(ctx, id); return err }},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := tc.read(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkImageBoundary(b *testing.B) {
	for _, count := range []int{300, 3000, 15000} {
		b.Run(fmt.Sprintf("messages%d", count), func(b *testing.B) {
			s := imageBenchmarkStore(b)
			ctx, id := context.Background(), s.Session().ID
			picture, _ := media.New("shot.png", "image/png", "png", 8, 8, []byte("small compressed sample"))
			messages := make([]modelapi.Message, count)
			entries := make([]runrecord.Entry, count)
			for i := range messages {
				messages[i] = modelapi.Message{Role: "assistant", Content: "completed"}
				if i%3 == 0 {
					messages[i] = modelapi.Message{Role: "user", Content: "check image"}
				}
				if i%3 == 1 {
					messages[i] = modelapi.Message{Role: "tool", ToolName: "read_image", ToolCallID: fmt.Sprintf("call%d", i), Images: []media.Image{picture}}
				}
				entries[i].Message = &messages[i]
			}
			if err := s.Commit(ctx, id, entries); err != nil {
				b.Fatal(err)
			}
			mark, err := s.MemoryWatermark(ctx, id, 0)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				boundary, err := s.CompactionBoundary(ctx, id, mark.Timeline, 0)
				if err != nil || boundary != mark.Timeline {
					b.Fatalf("boundary=%d: %v", boundary, err)
				}
			}
		})
	}
}
