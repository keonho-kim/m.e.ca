// Purpose: Preserve replay ordering across physical ring wrap and byte-budget eviction.
// Pattern: Bounded event log regression test.
// Usage: go test ./internal/commands/server -run TestRingReplay.
// Related: internal/commands/server/events.go.
package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
)

func TestRingReplayPreservesOrderAcrossWrapAndEviction(t *testing.T) {
	for _, size := range []int{0, maxTextBytes} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			s, _ := projectionFixture()
			const total = maxEvents + 100
			cursor := ""
			for i := range total {
				s.publish(Event{Record: record.Event{Kind: "progress", Text: fmt.Sprintf("%d:", i) + strings.Repeat("x", size)}})
				if i == total-40 {
					cursor = s.feed.cursor()
				}
			}
			next := total - 39
			for next < total {
				batch, err := s.Replay(cursor)
				if err != nil || batch.Snapshot != nil || len(batch.Events) == 0 {
					t.Fatal("retained cursor lost", batch, err)
				}
				for _, event := range batch.Events {
					if !strings.HasPrefix(event.Record.Text, fmt.Sprintf("%d:", next)) {
						t.Fatal("ring reordered events", next, event.Record.Text[:min(16, len(event.Record.Text))])
					}
					next++
					cursor = event.Cursor
				}
			}
			if s.feed.count > maxEvents || s.feed.bytes > maxEventBytes {
				t.Fatal("ring exceeded retention bounds")
			}
		})
	}
}
