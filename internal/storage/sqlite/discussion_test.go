// Purpose: Restore structured public feedback and shared work without duplicating receiver bodies or losing UI metadata.
// Pattern: SQLite round-trip and reference integrity tests.
// Usage: go test ./internal/storage/sqlite -run TestDiscussion.
// Related: internal/storage/sqlite/team_messages.go, internal/storage/sqlite/team_payloads.go.
package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/storage/history"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestDiscussionMetadataAndReferencedBodiesRoundTrip(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	work, err := root.AppendTeamMessage(ctx, channel.Message{From: "Hana", Text: "Implement authentication", Declaration: true, Note: &runrecord.TeamNote{Work: &runrecord.TeamWork{Paths: []string{"src/auth"}, Contract: "locked accounts return 423", Revision: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	feedback, err := root.AppendTeamMessage(ctx, channel.Message{From: "Ren", Text: strings.Repeat("Preserve existing client behavior. ", 100), Discussion: true, Note: &runrecord.TeamNote{ReplyTo: work.Sequence, ReplyAuthor: "Hana", BasedOn: work.Sequence, Mentions: []string{"Hana"}}})
	if err != nil {
		t.Fatal(err)
	}
	message := feedback.Context()
	if err := child.CommitContext(ctx, child.session.ID, []runrecord.Entry{{Message: &message}}, feedback.Sequence); err != nil {
		t.Fatal(err)
	}
	loaded, err := child.ContextMessages(ctx, child.session.ID, 0, 16<<20, false, nil)
	if err != nil || len(loaded) != 1 || loaded[0].Content != message.Content {
		t.Fatal("public reference hash or content changed", loaded, err)
	}
	page, err := root.TeamMessages(ctx, 0, feedback.Sequence, 10)
	if err != nil || len(page) != 2 || page[0].Note.Work.Contract != "locked accounts return 423" || page[1].Note.ReplyAuthor != "Hana" || !page[1].Discussion || !page[1].Attention("Hana") || page[1].Attention("Arjun") {
		t.Fatal("collaboration provenance changed", page, err)
	}
	visible, _, err := root.TimelineSummaries(ctx, root.session.ID, 0, 10, history.ReadBounds{})
	if err != nil || len(visible) != 2 || visible[1].Event.Team == nil || visible[1].Event.Team.ReplyTo != work.Sequence {
		t.Fatal("display hydration omitted public metadata", visible, err)
	}
}
