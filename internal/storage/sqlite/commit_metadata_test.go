// Purpose: Preserve first-user summaries, metadata and cursors across grouped writes and rollback.
// Pattern: Durable journal behavior regression test.
// Usage: SQLite tests verify commit-scoped session metadata.
// Related: internal/storage/sqlite/commit.go, internal/storage/sqlite/messages.go.
package sqlite

import (
	"context"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestCommitContextPreservesFirstRequestAndRollbackMetadata(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	first := modelapi.Message{Role: "user", Content: "first request"}
	second := modelapi.Message{Role: "user", Content: "second request"}
	if err := s.CommitContext(ctx, id, []runrecord.Entry{{Message: &first}, {Message: &second}}, 10); err != nil {
		t.Fatal(err)
	}
	var summary, updated string
	var cursor int64
	read := func() {
		t.Helper()
		if err := s.db.QueryRowContext(ctx, "SELECT summary,updated_at,context_cursor FROM sessions WHERE id=?", id).Scan(&summary, &updated, &cursor); err != nil {
			t.Fatal(err)
		}
	}
	read()
	if summary != "first request" || cursor != 10 {
		t.Fatal("first request or cursor lost", summary, cursor)
	}
	previous := updated
	invalid := runrecord.Event{Kind: "usage"}
	if err := s.CommitContext(ctx, id, []runrecord.Entry{{Message: &second}, {Event: &invalid}}, 20); err == nil {
		t.Fatal("invalid group committed")
	}
	read()
	if summary != "first request" || cursor != 10 || updated != previous {
		t.Fatal("failed group changed session metadata")
	}
}
