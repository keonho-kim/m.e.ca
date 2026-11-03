// Purpose: Coalesce telemetry without splitting text streams or crossing final usage barriers.
// Pattern: Observable bounded-mailbox regression test.
// Usage: Session delivery tests exercise slow-consumer usage and text interleaving.
// Related: internal/tui/session/delivery.go, internal/tui/session/request_delivery.go.
package session

import (
	"context"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestUsageDeliveryCoalescesWithoutBreakingText(t *testing.T) {
	for _, team := range []bool{false, true} {
		d := requestDelivery{team: team}
		queue := make(chan notice, 64)
		ctx := context.Background()
		for i := range 20 {
			if err := d.stream(ctx, queue, runrecord.Event{Kind: "stream", Agent: "agent-a", Text: string(rune('a' + i))}); err != nil {
				t.Fatal(err)
			}
			e := runrecord.Event{Kind: "usage_progress", Agent: "agent-a", Usage: &modelapi.TokenUsage{Output: int64(i)}}
			if err := d.post(ctx, queue, notice{Event: &e}); err != nil {
				t.Fatal(err)
			}
		}
		if len(queue) != 2 {
			t.Fatalf("team=%v: retained %d notices instead of two latest snapshots", team, len(queue))
		}
		for range 2 {
			e := (<-queue).Stream.take()
			if e == nil || e.Kind == "stream" && e.Text != "t" || e.Kind == "usage_progress" && e.Usage.Output != 19 {
				t.Fatal("latest text or measurement lost", e)
			}
		}
		first := runrecord.Event{Kind: "usage_progress", Usage: &modelapi.TokenUsage{Output: 20}}
		final := runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Output: 20}}
		d.post(ctx, queue, notice{Event: &first})
		d.post(ctx, queue, notice{Event: &final})
		first.Usage = &modelapi.TokenUsage{Output: 1}
		d.post(ctx, queue, notice{Event: &first})
		if (<-queue).Stream.take().Usage.Output != 20 || (<-queue).Event.Kind != "usage" || (<-queue).Stream.take().Usage.Output != 1 {
			t.Fatal("measurements crossed a final usage boundary")
		}
	}
}
