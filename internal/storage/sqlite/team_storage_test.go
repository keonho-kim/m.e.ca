// Purpose: Verify public-body sharing, receive-cursor atomicity and independent prompt/state ownership.
// Pattern: Temporary SQLite integration tests.
// Usage: go test ./internal/storage/sqlite -run TestTeamStorage.
// Related: internal/storage/sqlite/team_payloads.go, internal/storage/sqlite/commit.go.
package sqlite

import (
	"context"
	teamchannel "github.com/keonho-kim/meca/internal/agent/team/channel"
	"path/filepath"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func teamStorage(t *testing.T) (*Store, *Store) {
	t.Helper()
	ctx := context.Background()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	root, err := catalog.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	id, err := root.CreateTeamAgent(ctx)
	if err != nil {
		t.Fatal(err)
	}
	child, err := root.OpenTeamAgent(ctx, id.Session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { child.Close() })
	return root, child
}
func TestTeamStorageReferencesPublicBodyAndAuthenticatesIt(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	message, err := root.AppendTeamMessage(ctx, teamchannel.Message{From: "agent-a", Text: strings.Repeat("shared result ", 12000)})
	if err != nil {
		t.Fatal(err)
	}
	contextMessage := message.Context()
	if err := child.CommitContext(ctx, child.Session().ID, []runrecord.Entry{{Message: &contextMessage}}, message.Sequence); err != nil {
		t.Fatal(err)
	}
	var codec, size int
	if err := child.db.QueryRow("SELECT codec,length(body) FROM payloads").Scan(&codec, &size); err != nil || codec != 3 || size > 256 {
		t.Fatalf("private payload was copied: codec=%d bytes=%d err=%v", codec, size, err)
	}
	restored, err := child.ContextMessages(ctx, child.Session().ID, 0, 16*1024*1024, false)
	if err != nil || len(restored) != 1 || restored[0].Content != contextMessage.Content {
		t.Fatalf("restored=%d err=%v", len(restored), err)
	}
	records, _, err := child.MemoryRecords(ctx, child.Session().ID, 0, 1<<30)
	if err != nil || len(records) != 1 || records[0].Role != "peer" {
		t.Fatalf("peer became a user requirement: %+v %v", records, err)
	}
	if _, err := root.db.Exec("DELETE FROM tool_events WHERE kind='team_message'"); err != nil {
		t.Fatal(err)
	}
	if _, err := child.ContextMessages(ctx, child.Session().ID, 0, 16*1024*1024, false); err == nil {
		t.Fatal("missing public original was silently accepted")
	}
}
func TestTeamStorageReceiveCursorCommitsWithMessages(t *testing.T) {
	_, child := teamStorage(t)
	ctx := context.Background()
	id := child.Session().ID
	valid := modelapi.Message{Role: "user", Content: "A current user instruction"}
	invalid := runrecord.Event{Kind: "usage"}
	if err := child.CommitContext(ctx, id, []runrecord.Entry{{Message: &valid}, {Event: &invalid}}, 7); err == nil {
		t.Fatal("invalid batch accepted")
	}
	if cursor, err := child.ContextCursor(ctx, id); err != nil || cursor != 0 {
		t.Fatalf("cursor advanced after rollback: %d %v", cursor, err)
	}
	if messages, err := child.ContextMessages(ctx, id, 0, 16*1024*1024, false); err != nil || len(messages) != 0 {
		t.Fatal("message escaped rollback")
	}
	if err := child.CommitContext(ctx, id, []runrecord.Entry{{Message: &valid}}, 7); err != nil {
		t.Fatal(err)
	}
	if cursor, err := child.ContextCursor(ctx, id); err != nil || cursor != 7 {
		t.Fatal("cursor was not persisted")
	}
}
