// Purpose: Verify targeted invitations and independent private-session preparation.
// Pattern: Deterministic membership interleaving tests.
// Usage: go test -race ./internal/agent/team.
// Related: internal/agent/team/membership.go, internal/agent/team/invitation_test.go.
package team

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"github.com/keonho-kim/meca/internal/agent/team/source"
	"github.com/keonho-kim/meca/internal/diagnostics"
)

func TestSharedProposalConnectsInvitationsAndDeclaredDomains(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	ref, err := a.Discuss(r.ctx, "Review API compatibility", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	claimed := source.WorkDomain{Domain: "api-check", WorkRef: ref}
	other := source.WorkDomain{Domain: "storage-check"}
	if _, err := b.SetWork("Review API and storage", nil, "", 0, claimed, other); err != nil {
		t.Fatal(err)
	}
	invite := Invitation{Domain: "api-review", WorkRef: ref, Message: "Review the API"}
	if status, err := a.AddMember(r.ctx, invite); err != nil || status["status"] != "domain_active" || status["member_name"] != "Ren" {
		t.Fatalf("same proposal was not linked to its existing owner: %v %v", status, err)
	}
	if _, err := a.SetWork("Review API", nil, "", 0, source.WorkDomain{Domain: "api-review", WorkRef: ref}); err == nil {
		t.Fatal("concurrent declaration claimed another peer's proposal")
	}
	if scope, err := b.SetWork("Update the review", nil, "", 0); err != nil || len(scope.Domains) != 2 {
		t.Fatalf("omitted domains lost earlier claims: %+v %v", scope, err)
	}
	if _, err := b.SetWork("Release the API review", nil, "", 0, other); err != nil {
		t.Fatal(err)
	}
	if status, err := a.AddMember(r.ctx, invite); err != nil || status["status"] != "created" {
		t.Fatalf("released proposal could not be invited: %v %v", status, err)
	}
	if status, err := a.AddMember(r.ctx, Invitation{Domain: "api-third-label", WorkRef: ref, Message: "Review the API"}); err != nil || status["status"] != "domain_active" {
		t.Fatalf("different label duplicated a live invitation: %v %v", status, err)
	}
}

func TestInvitationLeavesUnrelatedTeammatesIdle(t *testing.T) {
	r := testRoom(t)
	const existing = 32
	r.options.MaxModelCalls = 64
	for i := 2; i < existing; i++ {
		member, err := r.options.Store.CreateTeamAgent(r.ctx)
		if err != nil {
			t.Fatal(err)
		}
		r.reserve(member, nil)
	}
	inviter := &Peer{ID: "Hana", room: r}
	result, err := inviter.AddMember(r.ctx, Invitation{Domain: "review", Message: "Review the API"})
	if err != nil || result["status"] != "created" {
		t.Fatalf("invite failed: %v %v", result, err)
	}
	r.mu.Lock()
	signalled := 0
	for _, peer := range r.active {
		if len(peer.wake) != 0 || peer.attention != 0 {
			signalled++
		}
	}
	r.mu.Unlock()
	if signalled != 0 {
		t.Fatalf("%d unrelated teammates received a wake signal", signalled)
	}
	t.Logf("one invitation with %d existing teammates sent %d wake signals", existing, signalled)
	messages, err := (&Peer{ID: "Ren", room: r}).ReadMessages(r.ctx, 0)
	if err != nil || len(messages) != 1 {
		t.Fatalf("invitation must remain public: %v %v", messages, err)
	}
}

func TestInvitationTimingUsesOneIdentifierWithoutWorkText(t *testing.T) {
	r := testRoom(t)
	var logs bytes.Buffer
	sink, err := diagnostics.New(diagnostics.Options{Writer: &logs, Level: slog.LevelDebug, Redact: func(s string) string { return s }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sink.Close() })
	r.runID = "run_invitation-test"
	r.ctx = diagnostics.WithSink(r.ctx, sink)
	p := &Peer{ID: "Hana", room: r}
	result, err := p.AddMember(r.ctx, Invitation{Domain: "private-work-key", Message: "private-work-body"})
	if err != nil || result["status"] != "created" {
		t.Fatalf("invite failed: %v %v", result, err)
	}
	peer := &Peer{ID: result["member_name"].(string), room: r}
	if allowed, err := peer.AdmitModel(t.Context()); !allowed || err != nil {
		t.Fatal("invitee could not use its first call", err)
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, line := range bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n")) {
		if bytes.Contains(line, []byte("private-work-")) {
			t.Fatal("invitation content entered operational logs")
		}
		var entry struct {
			Message  string `json:"msg"`
			Phase    string `json:"operation"`
			RunID    string `json:"run_id"`
			InviteID uint64 `json:"invitation_id"`
			Duration *int64 `json:"duration_ms"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatal(err)
		}
		if entry.Message != "team_invitation_phase" {
			continue
		}
		if entry.RunID != r.runID || entry.InviteID != 1 || entry.Duration == nil || *entry.Duration < 0 {
			t.Fatalf("uncorrelated phase timing: %s", line)
		}
		seen[entry.Phase] = true
	}
	for _, phase := range []string{"invite_reserve", "invite_prepare", "invite_activate", "invite_first_model"} {
		if !seen[phase] {
			t.Fatalf("missing timing for %s: %s", phase, &logs)
		}
	}
}

type independentRegistration struct {
	*memoryRoom
	calls                      atomic.Int32
	firstEntered, releaseFirst chan struct{}
}

func (s *independentRegistration) CreateTeamAgent(ctx context.Context) (identity.Member, error) {
	if s.calls.Add(1) == 1 {
		close(s.firstEntered)
		select {
		case <-ctx.Done():
			return identity.Member{}, ctx.Err()
		case <-s.releaseFirst:
		}
	}
	return s.memoryRoom.CreateTeamAgent(ctx)
}

func TestIndependentInvitationProceedsDuringPrivatePreparation(t *testing.T) {
	r := testRoom(t)
	store := &independentRegistration{
		memoryRoom: r.options.Store.(*memoryRoom), firstEntered: make(chan struct{}), releaseFirst: make(chan struct{}),
	}
	r.options.Store = store
	wait, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()
	send := func(name, domain string) <-chan error {
		done := make(chan error, 1)
		go func() {
			p := &Peer{ID: name, room: r}
			result, err := p.AddMember(wait, Invitation{Domain: domain, Message: "Inspect the component"})
			if err == nil && result["status"] != "created" {
				err = fmt.Errorf("unexpected invitation status: %v", result)
			}
			done <- err
		}()
		return done
	}
	first := send("Hana", "first-component")
	defer func() {
		close(store.releaseFirst)
		if err := <-first; err != nil && wait.Err() == nil {
			t.Error(err)
		}
	}()
	select {
	case <-store.firstEntered:
	case <-wait.Done():
		t.Fatal("first invitation did not enter private preparation")
	}
	duplicate := &Peer{ID: "Ren", room: r}
	result, err := duplicate.AddMember(wait, Invitation{Domain: "first-component", Message: "Inspect the same component"})
	if err != nil || result["status"] != "invitation_pending" {
		t.Fatalf("pending first invitation did not protect its work: %v %v", result, err)
	}
	select {
	case err := <-send("Ren", "second-component"):
		if err != nil {
			t.Fatal(err)
		}
	case <-wait.Done():
		t.Fatal("independent invitation waited for the first private session")
	}
}

func TestNewWorkClaimBeforeActivationDiscardsPreparedPeer(t *testing.T) {
	r := testRoom(t)
	store := &independentRegistration{
		memoryRoom: r.options.Store.(*memoryRoom), firstEntered: make(chan struct{}), releaseFirst: make(chan struct{}),
	}
	r.options.Store = store
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	wait, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		result, err := a.AddMember(wait, Invitation{Domain: "api-review", Message: "Review the API"})
		if err == nil && (result["status"] != "domain_active" || result["member_name"] != "Ren") {
			err = fmt.Errorf("prepared invitation bypassed a new claim: %v", result)
		}
		done <- err
	}()
	select {
	case <-store.firstEntered:
	case <-wait.Done():
		t.Fatal("invitation did not start preparing")
	}
	_, claimErr := b.SetWork("Review the API", nil, "", 0, source.WorkDomain{Domain: "api-review"})
	close(store.releaseFirst)
	if err := <-done; err != nil || claimErr != nil {
		t.Fatalf("claim or invitation failed: %v %v", claimErr, err)
	}
	members, err := store.TeamMembers(wait, "", 8)
	if err != nil || len(members) != 2 || r.reservedModelCalls != 0 {
		t.Fatalf("prepared session or model reservation leaked: %v %v", members, err)
	}
}
