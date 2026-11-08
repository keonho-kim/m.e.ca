// Purpose: Verify inbox admission drains a partial cached page through the latest user request.
// Pattern: Receive cursor regression test.
// Usage: go test ./internal/agent/team -run TestPublicHistoryInbox.
// Related: internal/agent/team/inbox.go, internal/agent/team/channel/history.go.
package team

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestPublicHistoryInboxDrainsPartialWindowThroughLatestUser(t *testing.T) {
	store := &memoryRoom{}
	ctx := context.Background()
	room := &Room{ctx: ctx, options: Options{Store: store}, latestPublic: 150}
	for i := 1; i <= 150; i++ {
		message, err := store.AppendTeamMessage(ctx, channel.Message{From: "user", Text: "latest instruction"})
		if err != nil {
			t.Fatal(err)
		}
		if i <= 50 {
			room.public.Append(message, int64(i-1))
		}
	}
	peer := &Peer{ID: "Hana", room: room}
	batch, err := peer.Inbox(ctx)
	if err != nil || len(batch.Messages) != 150 || batch.Cursor != 150 || peer.UserSequence != 150 {
		t.Fatalf("latest user was delayed behind a short cached page: %d cursor=%d %v", len(batch.Messages), batch.Cursor, err)
	}
}
