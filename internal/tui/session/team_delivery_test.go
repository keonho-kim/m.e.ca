// Purpose: Verify queued peer streams cannot overwrite each other or cross ordered approval barriers.
// Pattern: Deterministic mailbox behavior tests.
// Usage: go test -race ./internal/tui/session -run TestTeamDelivery.
// Related: internal/tui/session/request_delivery.go, internal/tui/session/delivery.go.
package session

import (
	"context"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestTeamDeliveryKeepsLatestSnapshotForEachPeer(t *testing.T) {
	delivery := requestDelivery{team: true}
	queue := make(chan notice, 8)
	for _, event := range []runrecord.Event{{Kind: "stream", Agent: "agent-a", Text: "a1"}, {Kind: "stream", Agent: "agent-b", Text: "b1"}, {Kind: "stream", Agent: "agent-a", Text: "a2"}} {
		if err := delivery.stream(context.Background(), queue, event); err != nil {
			t.Fatal(err)
		}
	}
	if len(queue) != 2 {
		t.Fatalf("mailboxes=%d", len(queue))
	}
	seen := map[string]string{}
	for range 2 {
		event := (<-queue).Stream.take()
		seen[event.Agent] = event.Text
	}
	if seen["agent-a"] != "a2" || seen["agent-b"] != "b1" {
		t.Fatalf("crossed sources: %v", seen)
	}
}
func TestTeamDeliveryFreezesStreamsBeforeApproval(t *testing.T) {
	delivery := requestDelivery{team: true}
	queue := make(chan notice, 8)
	ctx := context.Background()
	if err := delivery.stream(ctx, queue, runrecord.Event{Kind: "stream", Agent: "agent-a", Text: "before"}); err != nil {
		t.Fatal(err)
	}
	if err := delivery.post(ctx, queue, notice{Approval: &toolapi.Approval{Name: "change"}}); err != nil {
		t.Fatal(err)
	}
	if err := delivery.stream(ctx, queue, runrecord.Event{Kind: "stream", Agent: "agent-a", Text: "after"}); err != nil {
		t.Fatal(err)
	}
	if event := (<-queue).Stream.take(); event.Text != "before" {
		t.Fatal("later stream crossed approval")
	}
	if (<-queue).Approval == nil {
		t.Fatal("approval lost")
	}
	if event := (<-queue).Stream.take(); event.Text != "after" {
		t.Fatal("later stream lost")
	}
}
