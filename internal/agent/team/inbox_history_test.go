// Purpose: Verify inbox admission drains a partial cached page through the latest user request.
// Pattern: Receive cursor regression test.
// Usage: go test ./internal/agent/team -run TestPublicHistoryInbox.
// Related: internal/agent/team/inbox.go, internal/agent/team/channel/history.go.
package team

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
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

func TestInboxBudgetsImagesAcrossPublicPages(t *testing.T) {
	dir := t.TempDir()
	catalog, err := sqlite.OpenSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	store, err := catalog.Create(t.Context(), dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	r := &Room{ctx: t.Context(), options: Options{Store: store}}
	for i := range 6 {
		data := make([]byte, 4<<20)
		data[0] = byte(i)
		image, err := media.New("fixture.png", "image/png", "png", 1, 1, data)
		if err != nil {
			t.Fatal(err)
		}
		value, err := store.AppendTeamMessage(t.Context(), channel.Message{From: "user", Text: "Inspect image", Images: []media.Image{image}})
		if err != nil {
			t.Fatal(err)
		}
		r.public.Append(value, r.latestPublic)
		r.latestPublic = value.Sequence
	}
	peer := &Peer{ID: "Hana", room: r}
	received := 0
	for peer.Cursor < r.latestPublic {
		batch, err := peer.Inbox(t.Context())
		if err != nil || len(batch.Messages) == 0 {
			t.Fatalf("inbox did not make progress: %v", err)
		}
		received += len(batch.Messages)
		size := 0
		for _, message := range batch.Messages {
			size += message.ContextBytes()
		}
		if size > channel.PageBytes {
			t.Fatalf("batch of %d messages exceeded the context page budget: %d", len(batch.Messages), size)
		}
	}
	if received != 6 {
		t.Fatalf("received %d images, want 6", received)
	}
}
