// Purpose: Remove abandoned invitation state while preserving existing or used private sessions.
// Pattern: Temporary SQLite lifecycle regression tests.
// Usage: Storage tests exercise exact registration cleanup without application state.
// Related: internal/storage/sqlite/team_registration.go, internal/storage/sqlite/team_integrity_test.go.
package sqlite

import (
	"context"
	"os"
	"testing"

	model "github.com/keonho-kim/meca/internal/agent/model"
	record "github.com/keonho-kim/meca/internal/agent/record"
)

func TestDiscardUnstartedTeamSessionRemovesOnlyItsDatabase(t *testing.T) {
	root, existing := teamStorage(t)
	ctx := context.Background()
	member, err := root.CreateTeamAgent(ctx)
	if err != nil {
		t.Fatal(err)
	}
	child, err := root.OpenTeamAgent(ctx, member.Session)
	if err != nil {
		t.Fatal(err)
	}
	path := child.path
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}
	wrong := member
	wrong.ID = "Unregistered"
	if err := root.DiscardTeamAgent(ctx, wrong); err == nil {
		t.Fatal("mismatched registration was removed")
	}
	if err := root.DiscardTeamAgent(ctx, member); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if _, err := os.Stat(path + suffix); !os.IsNotExist(err) {
			t.Fatal("abandoned session file remains", path+suffix, err)
		}
	}
	if _, err := root.TeamMember(ctx, member.ID); err == nil {
		t.Fatal("abandoned registration remains")
	}
	if _, err := os.Stat(existing.path); err != nil {
		t.Fatal("existing session was affected", err)
	}
}

func TestDiscardRefusesPrivateConversationHistory(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	message := model.Message{Role: "user", Content: "Keep this history"}
	if err := child.Commit(ctx, child.session.ID, []record.Entry{{Message: &message}}); err != nil {
		t.Fatal(err)
	}
	members, err := root.TeamMembers(ctx, "", 10)
	if err != nil || len(members) != 1 {
		t.Fatal(err)
	}
	if err := root.DiscardTeamAgent(ctx, members[0]); err == nil {
		t.Fatal("a recorded conversation was discarded")
	}
	if _, err := root.TeamMember(ctx, members[0].ID); err != nil {
		t.Fatal("existing registration was removed", err)
	}
}
