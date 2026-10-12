// Purpose: Verify paged terminal history preserves the order of messages and tool events.
// Pattern: Repository behavior tests.
// Usage: go test ./internal/storage/sqlite -run TestTimeline.
// Related: internal/storage/sqlite/timeline.go.
package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/run"
)

func TestTimeline(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "v.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	root := t.TempDir()
	session, err := s.RestoreSession(ctx, root, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.AppendMessage(ctx, session.ID, agent.Message{Role: "user", Content: "fix"}); err != nil {
		t.Fatal(err)
	}
	if err = s.RecordEvent(ctx, session.ID, run.Event{Kind: "tool_started", Text: "Reading file"}); err != nil {
		t.Fatal(err)
	}
	if err = s.AppendMessage(ctx, session.ID, agent.Message{Role: "assistant", Content: "done"}); err != nil {
		t.Fatal(err)
	}
	rows, cursor, err := s.Timeline(ctx, session.ID, 0, 2)
	if err != nil || len(rows) != 2 || rows[0].Kind != "event" || rows[1].Kind != "message" {
		t.Fatalf("page: %+v %v", rows, err)
	}
	rows, _, err = s.Timeline(ctx, session.ID, cursor, 2)
	if err != nil || len(rows) != 1 || rows[0].Message.Content != "fix" {
		t.Fatalf("older page: %+v %v", rows, err)
	}
}
