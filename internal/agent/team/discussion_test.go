// Purpose: Preserve horizontal public feedback while routing attention independently of ownership and visibility.
// Pattern: Deterministic room contract tests.
// Usage: go test -race ./internal/agent/team -run TestDiscussion.
// Related: internal/agent/team/discussion.go, internal/agent/team/channel/attention.go.
package team

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/identity"
)

func TestDiscussionIsPublicAndDoesNotAssignWork(t *testing.T) {
	r := testRoom(t)
	third := identity.Member{ID: "Arjun", Session: "session_third"}
	r.options.Store.(*memoryRoom).members = append(r.options.Store.(*memoryRoom).members, third)
	r.reserve(third)
	hana, ren, arjun := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}, &Peer{ID: "Arjun", room: r}
	work, err := hana.SetWork("Login errors", []string{"/project/auth"}, "Return 403 for locked accounts", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, peer := range []*Peer{hana, ren, arjun} {
		if _, err := peer.Inbox(r.ctx); err != nil {
			t.Fatal(err)
		}
		peer.BeginDecision()
	}
	sequence, err := ren.Discuss(r.ctx, "The existing API uses 423; consider preserving it.", work.Sequence, []string{"Hana"})
	if err != nil {
		t.Fatal(err)
	}
	if r.countLocked() != 3 || !ren.HasWorkPaths(nil) || r.work.Scope("Hana").Contract != work.Contract {
		t.Fatal("feedback assigned work, changed a proposal or started a session")
	}
	public, err := arjun.Inbox(r.ctx)
	if err != nil || len(public.Messages) != 1 || public.Reconsider || !strings.Contains(public.Messages[0].Content, "423") {
		t.Fatal("unmentioned peer lost visibility or was forced to reason", public, err)
	}
	addressed, err := hana.Inbox(r.ctx)
	if err != nil || !addressed.Reconsider || len(addressed.Messages) != 1 {
		t.Fatal("addressed feedback did not request a decision", addressed, err)
	}
	if err := hana.CheckWork([]string{"/project/auth/login.go"}); err != nil {
		t.Fatal("a disagreement blocked an independent edit", err)
	}
	if _, err := arjun.Discuss(r.ctx, "I will check the existing clients to compare both options.", sequence, nil); err != nil {
		t.Fatal("a third teammate could not join the discussion", err)
	}
	hana.BeginDecision()
	if next, err := hana.Inbox(r.ctx); err != nil || next.Reconsider {
		t.Fatal("an unrelated reply or consumed hint forced another decision", next, err)
	}
}

func TestDiscussionValidatesRoomReferencesWithoutSideEffects(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	for _, args := range []struct {
		reply    int64
		mentions []string
	}{
		{reply: -1}, {reply: 1000}, {mentions: []string{"unknown"}},
	} {
		if _, err := p.Discuss(r.ctx, "Consider an alternative", args.reply, args.mentions); err == nil {
			t.Fatal("invalid public reference was accepted", args)
		}
	}
	if r.latestPublic != 0 || r.countLocked() != 2 {
		t.Fatal("invalid feedback changed the room")
	}
	closed := identity.Member{ID: "Arjun", Session: "session_closed"}
	r.options.Store.(*memoryRoom).members = append(r.options.Store.(*memoryRoom).members, closed)
	if _, err := p.Discuss(r.ctx, "This finding may help a later review.", 0, []string{"Arjun"}); err != nil {
		t.Fatal(err)
	}
	if r.active["Arjun"] != nil {
		t.Fatal("a mention restarted a closed session")
	}
}

func TestFailedWorkPublicationPreservesPreviousScopeAndContract(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	before, err := p.SetWork("Authentication", []string{"/project/auth"}, "Use 423", 0)
	if err != nil {
		t.Fatal(err)
	}
	r.options.Store = failedPublicStore{r.options.Store.(*memoryRoom)}
	if _, err := p.SetWork("New contribution", []string{"/project/other"}, "Changed contract", 0); err == nil {
		t.Fatal("failed public append was hidden")
	}
	after := r.work.Scope(p.ID)
	if after.Sequence != before.Sequence || after.Revision != before.Revision || after.Contract != before.Contract || after.Paths[0] != before.Paths[0] {
		t.Fatal("uncommitted work replaced the current contribution", after)
	}
}

func TestReleasedWorkRemainsDiscoverableWithoutReservingPaths(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	if _, err := p.SetWork("Authentication", []string{"/project/auth"}, "Use 423", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := p.SetWork("Implementation is ready for review", nil, "Use 423", 0); err != nil {
		t.Fatal(err)
	}
	page, err := (&Peer{ID: "Ren", room: r}).Members(r.ctx, "")
	if err != nil || len(page.Members) == 0 {
		t.Fatal(err)
	}
	work := page.Members[0].Work
	if work == nil || len(work.Paths) != 0 || work.Contract != "Use 423" || work.Summary != "Implementation is ready for review" {
		t.Fatal("released paths hid the peer's current result or retained ownership", work)
	}
}
