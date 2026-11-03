// Purpose: Verify atomic peer registration, short-message storage and exact concurrent public receipts.
// Pattern: Failure-injected temporary SQLite integration tests.
// Usage: go test -race ./internal/storage/sqlite -run TestTeam.
// Related: internal/storage/sqlite/team_registration.go, internal/storage/sqlite/team_messages.go.
package sqlite

import (
	"context"
	"fmt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	teamchannel "github.com/keonho-kim/meca/internal/agent/team/channel"
	"path/filepath"
	"sync"
	"testing"
)

func TestTeamRegistrationFailureRemovesOnlyNewDatabase(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	sessions, err := root.teamSessions()
	if err != nil {
		t.Fatal(err)
	}
	directory, _, err := sessions.projectDirectory(root.session.ProjectRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	before, err := filepath.Glob(filepath.Join(directory, "*.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := root.db.Exec(`CREATE TRIGGER fail_registration BEFORE INSERT ON team_members BEGIN SELECT RAISE(ABORT,'registration failed'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := root.CreateTeamAgent(ctx); err == nil {
		t.Fatal("registration failure was ignored")
	}
	after, err := filepath.Glob(filepath.Join(directory, "*.db"))
	if err != nil || len(before) != 1 || len(after) != 1 || before[0] != after[0] {
		t.Fatalf("orphan or lost database: %v -> %v (%v)", before, after, err)
	}
	members, err := root.TeamMembers(ctx, "", 10)
	if err != nil || len(members) != 1 || members[0].Session != child.session.ID {
		t.Fatal("existing peer registry changed")
	}
}
func TestTeamShortMessagesStayInline(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	value, err := root.AppendTeamMessage(ctx, teamchannel.Message{From: "agent-a", Text: "Check api.go."})
	if err != nil {
		t.Fatal(err)
	}
	message := value.Context()
	if err := child.CommitContext(ctx, child.session.ID, []runrecord.Entry{{Message: &message}}, value.Sequence); err != nil {
		t.Fatal(err)
	}
	var references int
	if err := child.db.QueryRow("SELECT COUNT(*) FROM payloads WHERE codec=3").Scan(&references); err != nil || references != 0 {
		t.Fatalf("short body got a larger reference: %d %v", references, err)
	}
	restored, err := child.ContextMessages(ctx, child.session.ID, 0, 1<<20, false)
	if err != nil || len(restored) != 1 || restored[0].Content != message.Content {
		t.Fatalf("short context changed: %v %v", restored, err)
	}
}
func TestTeamConcurrentPublicationsReturnOwnSequence(t *testing.T) {
	root, _ := teamStorage(t)
	ctx := context.Background()
	var workers sync.WaitGroup
	for i := range 16 {
		workers.Go(func() {
			text := fmt.Sprintf("result %d", i)
			value, err := root.AppendTeamMessage(ctx, teamchannel.Message{From: "agent-a", Text: text})
			if err != nil {
				t.Error(err)
				return
			}
			page, err := root.TeamMessages(ctx, value.Sequence-1, value.Sequence, 1)
			if err != nil || len(page) != 1 || page[0].Text != text {
				t.Errorf("incorrect publication receipt: %v %v", page, err)
			}
		})
	}
	workers.Wait()
}
