// Purpose: Keep private-session preparation independent of public conversation admission.
// Pattern: Controlled repository interleaving tests.
// Usage: Team tests block creation while inspecting publication and steering behavior.
// Related: internal/agent/team/membership.go, internal/agent/team/decision.go.
package team

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
)

func TestConcurrentInvitationsForOneDomainStartOnePeer(t *testing.T) {
	r := testRoom(t)
	var workers sync.WaitGroup
	for i, name := range []string{"Hana", "Ren"} {
		workers.Go(func() {
			peer := &Peer{ID: name, room: r}
			domain := []string{"API-review", "api-review"}[i]
			result, err := peer.AddMember(r.ctx, Invitation{Domain: domain, Message: fmt.Sprintf("Review the API from perspective %d", i)})
			if err != nil || (result["status"] != "created" && result["status"] != "domain_active" && result["status"] != "invitation_pending") {
				t.Errorf("invite result=%v error=%v", result, err)
			}
		})
	}
	workers.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.countLocked() != 3 || r.reservedModelCalls != 1 || len(r.options.Store.(*memoryRoom).members) != 3 {
		t.Fatal("duplicate domain admitted a distinct private session")
	}
}

func TestPeerExitBeforeFirstCallReleasesInvitation(t *testing.T) {
	r := testRoom(t)
	r.options.Execute = func(context.Context, identity.Member, string, *Peer) (run.Result, error) {
		return run.Result{}, errors.New("private session could not open")
	}
	p := &Peer{ID: "Hana", room: r}
	invitation := Invitation{Domain: "api-review", Message: "Review the API"}
	result, err := p.AddMember(r.ctx, invitation)
	if err != nil || result["status"] != "created" {
		t.Fatalf("invite failed: %v %v", result, err)
	}
	wait, cancel := context.WithTimeout(r.ctx, 3*time.Second)
	defer cancel()
	select {
	case <-r.changed:
	case <-wait.Done():
		t.Fatal("failed peer did not release", wait.Err())
	}
	r.mu.Lock()
	reserved := r.reservedModelCalls
	r.mu.Unlock()
	if reserved != 0 {
		t.Fatalf("abandoned model reservations = %d", reserved)
	}
	if result, err = p.AddMember(r.ctx, invitation); err != nil || result["status"] != "created" {
		t.Fatalf("domain was not released on exit: %v %v", result, err)
	}
}

func TestMemoryPressureDefersInvitationBeforeCreatingSession(t *testing.T) {
	r := testRoom(t)
	pressured := true
	r.options.MemoryPressure = func() (bool, error) { return pressured, nil }
	p := &Peer{ID: "Hana", room: r}
	invitation := Invitation{Domain: "api-review", Message: "Review the API"}
	result, err := p.AddMember(r.ctx, invitation)
	members, listErr := r.options.Store.TeamMembers(r.ctx, "", 10)
	if err != nil || listErr != nil || result["reason"] != "memory_pressure" || len(members) != 2 || r.reservedModelCalls != 0 {
		t.Fatalf("pressured invite changed the room: result=%v err=%v members=%v", result, err, members)
	}
	pressured = false
	if result, err = p.AddMember(r.ctx, invitation); err != nil || result["status"] != "created" {
		t.Fatalf("capacity was not available after pressure cleared: %v %v", result, err)
	}
}

func TestInvitationReservesFirstModelCallForNewPeer(t *testing.T) {
	r := testRoom(t)
	r.options.MaxModelCalls = 3
	inviter := &Peer{ID: "Hana", room: r}
	result, err := inviter.AddMember(r.ctx, Invitation{Domain: "api-review", Message: "Review the API"})
	if err != nil || result["status"] != "created" {
		t.Fatalf("invite failed: %v %v", result, err)
	}
	for range 2 {
		if allowed, err := inviter.AdmitModel(t.Context()); !allowed || err != nil {
			t.Fatal("inviter lost an unreserved model call", err)
		}
	}
	invitee := &Peer{ID: result["member_name"].(string), room: r}
	if allowed, err := invitee.AdmitModel(t.Context()); !allowed || err != nil {
		t.Fatal("invitee could not use the reserved call", err)
	}
	if allowed, err := inviter.AdmitModel(t.Context()); allowed || err != nil || !r.budgetExhausted || r.reservedModelCalls != 0 || r.modelCalls != 3 {
		t.Fatal("invitee could not use exactly the reserved call")
	}
}

func TestModelAdmissionResumesAfterPendingInvitationIsCancelled(t *testing.T) {
	r := testRoom(t)
	r.options.MaxModelCalls = 4
	store := &slowRegistration{memoryRoom: r.options.Store.(*memoryRoom), entered: make(chan struct{}), release: make(chan struct{})}
	r.options.Store = store
	prepare, cancelPrepare := context.WithCancel(t.Context())
	defer cancelPrepare()
	invitation := make(chan error, 1)
	go func() {
		_, err := (&Peer{ID: "Hana", room: r}).AddMember(prepare, Invitation{Domain: "parallel", Message: "Inspect the files"})
		invitation <- err
	}()
	<-store.entered
	peer := &Peer{ID: "Ren", room: r}
	for range 3 {
		if allowed, err := peer.AdmitModel(t.Context()); !allowed || err != nil {
			t.Fatal("lost unreserved model capacity", err)
		}
	}
	wait, cancelWait := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancelWait()
	admission := make(chan error, 1)
	go func() {
		allowed, err := peer.AdmitModel(wait)
		if err == nil && !allowed {
			err = errors.New("temporarily reserved capacity was reported as exhausted")
		}
		admission <- err
	}()
	for {
		r.mu.Lock()
		waiting, exhausted := r.modelBudgetChanged != nil, r.budgetExhausted
		r.mu.Unlock()
		if exhausted {
			t.Fatal("room marked its remaining reservation as exhausted")
		}
		if waiting {
			break
		}
		if wait.Err() != nil {
			t.Fatal("model admission did not wait for the reservation")
		}
		runtime.Gosched()
	}
	cancelPrepare()
	if err := <-invitation; !errors.Is(err, context.Canceled) {
		t.Fatal("invitation did not return cancellation", err)
	}
	if err := <-admission; err != nil {
		t.Fatal("waiting agent did not receive the returned call", err)
	}
}

func TestNinthTeammateStartsAndPublishesAResult(t *testing.T) {
	r := testRoom(t)
	wait, cancel := context.WithTimeout(r.ctx, 3*time.Second)
	defer cancel()
	p := &Peer{ID: "Hana", room: r}
	published := make(chan string, 7)
	r.options.Execute = func(ctx context.Context, member identity.Member, _ string, peer *Peer) (run.Result, error) {
		batch, err := peer.Inbox(ctx)
		if err != nil {
			return run.Result{}, err
		}
		peer.BeginDecision()
		_ = peer.ContextObservations(batch.Messages)
		if allowed, err := peer.AdmitModel(ctx); !allowed || err != nil {
			return run.Result{}, fmt.Errorf("invited peer %s received no first model call: %v", member.ID, err)
		}
		if err := peer.Emit(runrecord.Event{Kind: "progress", Text: "Verified by " + member.ID}); err != nil {
			return run.Result{}, err
		}
		published <- member.ID
		<-ctx.Done()
		return run.Result{}, ctx.Err()
	}
	var ninth string
	for i := range 7 {
		invitation := Invitation{Domain: fmt.Sprintf("component-%d", i), Message: "Review the component"}
		result, err := p.AddMember(r.ctx, invitation)
		if err != nil || result["status"] != "created" {
			t.Fatalf("invite %d failed: %v %v", i, result, err)
		}
		ninth = result["member_name"].(string)
	}
	for range 7 {
		select {
		case <-published:
		case <-wait.Done():
			t.Fatal("invited teammate did not publish", wait.Err())
		}
	}
	messages, err := p.ReadMessages(r.ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range messages {
		if message.From == ninth && message.Text == "Verified by "+ninth {
			return
		}
	}
	t.Fatalf("ninth teammate %s left no public result", ninth)
}

type slowRegistration struct {
	*memoryRoom
	entered, release chan struct{}
}

func TestInstructionDuringSessionCreationPreventsLaunchAndDiscardsNewState(t *testing.T) {
	r := testRoom(t)
	store := &slowRegistration{memoryRoom: r.options.Store.(*memoryRoom), entered: make(chan struct{}), release: make(chan struct{})}
	r.options.Store = store
	p := &Peer{ID: "Hana", room: r}
	done := make(chan error, 1)
	go func() {
		_, err := p.AddMember(r.ctx, Invitation{Domain: "authentication", Message: "Change authentication"})
		done <- err
	}()
	<-store.entered
	if _, err := r.publish(channel.Message{From: "user", Text: "Do not add another teammate"}); err != nil {
		close(store.release)
		<-done
		t.Fatal(err)
	}
	close(store.release)
	if err := <-done; err == nil {
		t.Fatal("obsolete invitation launched a peer")
	}
	members, err := store.TeamMembers(r.ctx, "", 100)
	if err != nil || len(members) != 2 || r.countLocked() != 2 || r.reservedModelCalls != 0 {
		t.Fatal("obsolete invitation retained unused session state", members, err)
	}
}

func (s *slowRegistration) CreateTeamAgent(ctx context.Context) (identity.Member, error) {
	close(s.entered)
	select {
	case <-s.release:
	case <-ctx.Done():
		return identity.Member{}, ctx.Err()
	}
	return s.memoryRoom.CreateTeamAgent(ctx)
}

func TestSessionCreationLeavesPublicJournalAvailable(t *testing.T) {
	r := testRoom(t)
	store := &slowRegistration{memoryRoom: r.options.Store.(*memoryRoom), entered: make(chan struct{}), release: make(chan struct{})}
	r.options.Store = store
	p := &Peer{ID: "Hana", room: r}
	done := make(chan error, 1)
	go func() {
		_, err := p.AddMember(r.ctx, Invitation{Domain: "file-inspection", Message: "Inspect an independent file"})
		done <- err
	}()
	<-store.entered
	available := r.logMu.TryLock()
	if available {
		r.logMu.Unlock()
	}
	close(store.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !available {
		t.Fatal("private-session preparation blocked public messages and user input")
	}
}
