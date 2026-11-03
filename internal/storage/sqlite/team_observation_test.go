// Purpose: Restore shared operation facts without private body copies or duplicate own wakeups.
// Pattern: Temporary SQLite context and display contract test.
// Usage: Storage tests verify team observations through existing journal tables.
// Related: internal/storage/sqlite/team_messages.go, internal/storage/sqlite/team_context_page.go.
package sqlite

import (
	"context"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestTeamObservationRestoresWithoutVisibleDuplicateOrOwnWakeup(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	value, err := root.AppendTeamMessage(ctx, channel.Message{From: "Hana", Text: "write_file changed api.go", Observation: true})
	if err != nil {
		t.Fatal(err)
	}
	if pending, err := root.TeamPending(ctx, 0, value.Sequence, "Hana"); err != nil || pending {
		t.Fatal("own operation restarted the member", err)
	}
	if pending, err := root.TeamPending(ctx, 0, value.Sequence, "Ren"); err != nil || !pending {
		t.Fatal("teammate missed actual work", err)
	}
	message := value.Context()
	if err := child.CommitContext(ctx, child.Session().ID, []runrecord.Entry{{Message: &message}}, value.Sequence); err != nil {
		t.Fatal(err)
	}
	messages, err := child.ContextMessages(ctx, child.Session().ID, 0, 16<<20, false)
	if err != nil || len(messages) != 1 || messages[0].Content != message.Content {
		t.Fatal("fact restoration changed", err)
	}
	public, err := root.TeamMessages(ctx, 0, value.Sequence, 10)
	if err != nil || len(public) != 1 || !public[0].Observation {
		t.Fatal("observation provenance lost", err)
	}
	visible, _, err := root.TimelineSummaries(ctx, root.Session().ID, 0, 10)
	if err != nil || len(visible) != 0 {
		t.Fatal("operation fact duplicated a visible tool card", err)
	}
}

func TestPassiveStatusAndWorkDeclarationKeepTheirAdmissionSemanticsAfterRestore(t *testing.T) {
	root, _ := teamStorage(t)
	ctx := context.Background()
	status, err := root.AppendTeamMessage(ctx, channel.Message{From: "Hana", Text: "Inspection exited successfully", Observation: true, Passive: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, own := range []string{"Hana", "Ren"} {
		if pending, err := root.TeamPending(ctx, 0, status.Sequence, own); err != nil || pending {
			t.Fatal("passive status resumed inference", own, err)
		}
	}
	work, err := root.AppendTeamMessage(ctx, channel.Message{From: "Hana", Text: "Own api/", Declaration: true})
	if err != nil {
		t.Fatal(err)
	}
	if pending, err := root.TeamPending(ctx, 0, work.Sequence, "Hana"); err != nil || pending {
		t.Fatal("native declaration returned to its author", err)
	}
	if pending, err := root.TeamPending(ctx, 0, work.Sequence, "Ren"); err != nil || !pending {
		t.Fatal("declaration did not notify a teammate", err)
	}
	values, err := root.TeamMessages(ctx, 0, work.Sequence, 10)
	if err != nil || len(values) != 2 || !values[0].Passive || !values[0].Observation || !values[1].Declaration {
		t.Fatal("stored admission semantics changed", values, err)
	}
	visible, _, err := root.TimelineSummaries(ctx, root.session.ID, 0, 10)
	if err != nil || len(visible) != 1 || visible[0].Event.Kind != "team_work" {
		t.Fatal("work declaration disappeared or hidden status gained a card", visible, err)
	}
}

func TestExecutionReferenceAndPublicFactCommitAtomically(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	if err := child.RecordEvent(ctx, child.session.ID, runrecord.Event{Kind: "tool_succeeded", Text: "created"}); err != nil {
		t.Fatal(err)
	}
	message := channel.Message{From: "Hana", Text: "write_file changed api.go", Observation: true,
		Reference: &runrecord.EventReference{Session: child.session.ID, Sequence: child.latestEvent.Load()}}
	if _, err := root.db.Exec(`CREATE TRIGGER reject_fact BEFORE INSERT ON tool_events
 WHEN NEW.kind='team_observation' BEGIN SELECT RAISE(ABORT,'fact rejected'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := root.AppendTeamMessage(ctx, message); err == nil {
		t.Fatal("failed observation was accepted")
	}
	var references int
	if err := root.db.QueryRow("SELECT COUNT(*) FROM tool_events WHERE kind='team_reference'").Scan(&references); err != nil || references != 0 {
		t.Fatal("original reference escaped the failed fact transaction", references, err)
	}
	if _, err := root.db.Exec("DROP TRIGGER reject_fact"); err != nil {
		t.Fatal(err)
	}
	value, err := root.AppendTeamMessage(ctx, message)
	if err != nil {
		t.Fatal(err)
	}
	var referenceSequence int64
	if err := root.db.QueryRow(`SELECT t.sequence FROM timeline t JOIN tool_events e ON t.source_sequence=e.sequence
 WHERE t.kind='event' AND e.kind='team_reference'`).Scan(&referenceSequence); err != nil || referenceSequence+1 != value.Sequence {
		t.Fatal("reference and observation were not adjacent committed records", referenceSequence, value.Sequence, err)
	}
	visible, _, err := root.TimelineSummaries(ctx, root.session.ID, 0, 10)
	if err != nil || len(visible) != 1 || visible[0].Event.Text != "created" {
		t.Fatal("original result did not restore through its single reference", visible, err)
	}
}
