// Purpose: Measure retained-event lookup and bounded tool projection.
// Pattern: Deterministic server read benchmarks.
// Usage: go test ./internal/commands/server -run '^$' -bench 'Benchmark(EventReplay|ToolProjection)' -benchmem.
// Related: internal/commands/server/events.go, internal/commands/server/projection.go.
package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

func BenchmarkEventReplay(b *testing.B) {
	for _, count := range []int{32, maxEvents} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			s, r := projectionFixture()
			for range count {
				s.emit(r, record.Event{Kind: "progress", Text: "update"})
			}
			cursor := s.feed.entry(s.feed.count - 2).value.Cursor
			b.ReportAllocs()
			for b.Loop() {
				batch, err := s.Next(b.Context(), cursor)
				if err != nil || len(batch.Events) != 1 {
					b.Fatal(batch, err)
				}
			}
		})
	}
}

func BenchmarkToolProjection(b *testing.B) {
	for _, size := range []int{64 << 10, 8 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			value := Event{Record: record.Event{Kind: "tool_succeeded", Call: &tool.Call{ID: "write", Name: "write_file", Arguments: []byte(`{"content":"` + strings.Repeat("x", size) + `"}`)}}}
			b.ReportAllocs()
			for b.Loop() {
				result := ProjectEvent(value)
				if len(result.Record.Call.Arguments) > maxTextBytes {
					b.Fatal("unbounded tool projection")
				}
			}
		})
	}
}

func BenchmarkEventPublishSteady(b *testing.B) {
	s := &Service{feed: newEventLog()}
	value := Event{Record: record.Event{Kind: "progress", Text: "update"}}
	for range maxEvents {
		s.publish(value)
	}
	b.ReportAllocs()
	for b.Loop() {
		s.publish(value)
	}
}
