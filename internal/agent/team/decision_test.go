// Purpose: Bind team effects and departures to instructions and public feedback actually reviewed.
// Pattern: Deterministic collaboration boundary regression tests.
// Usage: Team tests exercise admission, model review and lifecycle without providers.
// Related: internal/agent/team/decision.go, internal/agent/team/membership.go.
package team

import (
	"testing"

	record "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestReceivedInstructionStillNeedsANewModelDecision(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	r.publish(channel.Message{From: "user", Text: "Implement authentication"})
	p.Inbox(r.ctx)
	p.BeginDecision()
	r.publish(channel.Message{From: "user", Text: "Preserve auth.py"})
	p.Inbox(r.ctx)
	if err := p.CheckUserDecision(); err == nil {
		t.Fatal("receiving a user message was mistaken for reviewing it")
	}
	if _, err := p.SetWork("Overwrite authentication", []string{"/project/auth.py"}, "", 0); err == nil {
		t.Fatal("stale coordination changed work")
	}
	if _, err := p.AddMember(r.ctx, "", "Overwrite authentication"); err == nil {
		t.Fatal("stale coordination created a peer")
	}
	if _, err := p.Discuss(r.ctx, "I will overwrite authentication.", 0, nil); err == nil {
		t.Fatal("stale response was published after a new instruction")
	}
	p.BeginDecision()
	if err := p.CheckUserDecision(); err != nil {
		t.Fatal("fresh model decision remained blocked", err)
	}
}

func TestDepartureRequiresReviewOfAddressedFeedback(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	r.publish(channel.Message{From: "user", Text: "Implement authentication"})
	a.Inbox(r.ctx)
	a.BeginDecision()
	b.Inbox(r.ctx)
	b.BeginDecision()
	seq, err := a.Discuss(r.ctx, "The interface conflicts with existing clients; please clarify.", 0, []string{"Ren"})
	if err != nil {
		t.Fatal(err)
	}
	b.Inbox(r.ctx)
	if left, err := b.Leave(r.ctx, "Finished"); left || err == nil {
		t.Fatal("unreviewed feedback did not prevent departure", err)
	}
	b.BeginDecision()
	if _, err := b.Discuss(r.ctx, "I will preserve the existing client contract.", seq, nil); err != nil {
		t.Fatal(err)
	}
	if left, err := b.Leave(r.ctx, "Clarified the result"); err != nil || !left {
		t.Fatal("reviewed feedback prevented departure", err)
	}
}

func TestDiscussAnnouncementAndExplicitReplyEnableUsefulProgress(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	seq, err := a.Discuss(r.ctx, "Ren, I will inspect auth.py; which interface should I target?", 0, []string{"Ren"})
	if err != nil || !a.HasSpoken() {
		t.Fatal("committed public discussion did not count as speech", err)
	}
	a.Inbox(r.ctx)
	a.BeginDecision()
	b.Inbox(r.ctx)
	b.BeginDecision()
	if _, err := b.Discuss(r.ctx, "Use Authenticate(token); the interface is ready.", seq, nil); err != nil {
		t.Fatal(err)
	}
	batch, err := a.Inbox(r.ctx)
	if err != nil || !batch.Reconsider || len(batch.Messages) != 1 {
		t.Fatal("explicit answer did not resume the questioner", batch, err)
	}
	if err := b.Emit(record.Event{Kind: "progress", Text: "Continuing routine work."}); err != nil {
		t.Fatal(err)
	}
	a.BeginDecision()
	if next, err := a.Inbox(r.ctx); err != nil || next.Reconsider {
		t.Fatal("routine progress forced another decision", err)
	}
}

func TestRestoredPeerRecoversAlreadyReceivedUserSequence(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	r.publish(channel.Message{From: "user", Text: "Implement authentication"})
	p.Inbox(r.ctx)
	p.BeginDecision()
	restored := &Peer{ID: "Hana", room: r, Cursor: p.Cursor}
	batch, err := restored.Inbox(r.ctx)
	restored.BeginDecision()
	if err != nil || batch.UserSequence != r.latestUser || restored.CheckUserDecision() != nil {
		t.Fatal("restored peer lost the instruction already saved with its receive cursor", batch, err)
	}
}
