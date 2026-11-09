// Purpose: Limit additional inference to relevant work while preserving all peer observations.
// Pattern: Deterministic routing and public-visibility tests.
// Usage: Team tests compare receive content with reconsideration decisions.
// Related: internal/agent/team/attention.go, internal/agent/team/inbox.go.
package team

import (
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestUnrelatedChangesRemainVisibleWithoutRestartingInference(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	a.SetWork("API", []string{"/project/api"}, "", 0)
	b.SetWork("Docs", []string{"/project/docs"}, "", 0)
	b.Inbox(r.ctx)
	b.BeginDecision()
	for _, m := range []channel.Message{
		{From: "Hana", Text: "API changed", Observation: true, ChangedPaths: []string{"/project/api/router.go"}},
		{From: "Hana", Text: "Moved to API tests", Declaration: true},
	} {
		if m.Declaration {
			if _, err := a.SetWork("API tests", []string{"/project/api_tests"}, "", 0); err != nil {
				t.Fatal(err)
			}
		} else if _, err := r.publish(m); err != nil {
			t.Fatal(err)
		}
		batch, err := b.Inbox(r.ctx)
		if err != nil || len(batch.Messages) != 1 || batch.Reconsider {
			t.Fatal("unrelated work woke a model or lost visibility", batch, err)
		}
	}
	if _, err := r.publish(channel.Message{From: "Hana", Text: "Shared docs updated", Observation: true, ChangedPaths: []string{"/project/docs/api.md"}}); err != nil {
		t.Fatal(err)
	}
	batch, err := b.Inbox(r.ctx)
	if err != nil || !batch.Reconsider {
		t.Fatal("related changes did not resume a decision", err)
	}
	b.BeginDecision()
	if _, err := r.publish(channel.Message{From: "Hana", Text: "Unknown command failure", Observation: true}); err != nil {
		t.Fatal(err)
	}
	batch, err = b.Inbox(r.ctx)
	if err != nil || !batch.Reconsider {
		t.Fatal("unknown effects lost the conservative fallback", err)
	}
}
