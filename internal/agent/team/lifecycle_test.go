// Purpose: Verify cancellation, fatal delivery, request budgets and concurrent restoration.
// Pattern: Deterministic cooperative lifecycle tests.
// Usage: go test -race ./internal/agent/team.
// Related: internal/agent/team/room.go, internal/agent/team/failure.go.
package team

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/work"
)

func TestCancellationJoinsEveryPeerWithoutRestarting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, 2)
	var calls, closed atomic.Int32
	finished := make(chan error, 1)
	go func() {
		_, err := Run(ctx, Options{Store: &memoryRoom{}, Emit: quiet, MaxModelCalls: 10, MaxSeconds: 5,
			Execute: func(ctx context.Context, _ identity.Member, _ string, _ *Peer) (run.Result, error) {
				calls.Add(1)
				started <- struct{}{}
				<-ctx.Done()
				closed.Add(1)
				return run.Result{}, ctx.Err()
			}}, "Work on the request")
		finished <- err
	}()
	<-started
	<-started
	cancel()
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 2 || closed.Load() != 2 {
		t.Fatalf("started=%d closed=%d", calls.Load(), closed.Load())
	}
}

func TestSharedDeliveryFailureStopsTheWholeRequest(t *testing.T) {
	want := errors.New("public observation could not be delivered")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := Run(ctx, Options{Store: &memoryRoom{}, MaxModelCalls: 10, MaxSeconds: 5,
		Emit: func(event runrecord.Event) error {
			if event.Kind == "team_notice" {
				return want
			}
			return nil
		}, Execute: func(context.Context, identity.Member, string, *Peer) (run.Result, error) {
			t.Error("a peer started after its join notification failed")
			return run.Result{}, nil
		}}, "Build the requested change")
	if !errors.Is(err, want) || result.Status == runrecord.Answered {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

func TestModelAdmissionBudgetIsShared(t *testing.T) {
	var admitted atomic.Int32
	result, err := Run(context.Background(), Options{Store: &memoryRoom{}, Emit: quiet, MaxModelCalls: 1, MaxSeconds: 5,
		Execute: func(_ context.Context, _ identity.Member, _ string, peer *Peer) (run.Result, error) {
			if peer.AdmitModel() {
				admitted.Add(1)
			}
			return run.Result{Status: runrecord.Stopped, End: work.End{Reason: "model call budget exhausted"}}, nil
		}}, "Work within the request budget")
	if admitted.Load() != 1 || err == nil || !strings.Contains(err.Error(), "model call budget exhausted") || result.Status != runrecord.Stopped {
		t.Fatalf("admitted=%d result=%+v error=%v", admitted.Load(), result, err)
	}
}

func TestConcurrentRestorationStartsOnePrivateSession(t *testing.T) {
	r := testRoom(t)
	r.options.Store.(*memoryRoom).members = append(r.options.Store.(*memoryRoom).members, identity.Member{ID: "Arjun", Session: "session_abc"})
	p := &Peer{ID: "Hana", room: r}
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			if _, err := p.AddMember(r.ctx, "Arjun", "Check the remaining work"); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.countLocked() != 3 || len(r.options.Store.(*memoryRoom).members) != 3 {
		t.Fatalf("active=%d identities=%d", r.countLocked(), len(r.options.Store.(*memoryRoom).members))
	}
}

func TestPublishedContributionEntersOwnContextWithoutRepeatingAssistantResponses(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	if _, err := p.Leave(r.ctx, "I updated storage/query.go; keep the interface unchanged."); err != nil {
		t.Fatal(err)
	}
	batch, err := p.Inbox(r.ctx)
	if err != nil || len(batch.Messages) != 1 || !strings.Contains(batch.Messages[0].Content, "storage/query.go") {
		t.Fatalf("own contribution would be lost during compaction: %+v %v", batch, err)
	}
	_, err = r.publish(channel.Message{From: p.ID, Text: "A completed assistant response", FromStream: true})
	if err != nil {
		t.Fatal(err)
	}
	if batch, err = p.Inbox(r.ctx); err != nil || len(batch.Messages) != 0 || batch.Cursor == 0 {
		t.Fatalf("own native assistant response was repeated: %+v %v", batch, err)
	}
}

func TestNewUserInstructionPrecedesGoodbye(t *testing.T) {
	r := testRoom(t)
	pending := []string{"Only inspect the files; keep them unchanged."}
	r.options.PollSteering = func() []string {
		out := pending
		pending = nil
		return out
	}
	p := &Peer{ID: "Hana", room: r}
	if _, err := p.Leave(r.ctx, "The previous request is complete"); err == nil {
		t.Fatal("peer left without reading the newest instruction")
	}
	if r.active[p.ID].leaving {
		t.Fatal("failed goodbye changed membership")
	}
	batch, err := p.Inbox(r.ctx)
	if err != nil || batch.Objective != "Only inspect the files; keep them unchanged." {
		t.Fatalf("new instruction missing: %+v %v", batch, err)
	}
	p.BeginDecision()
	if departed, err := p.Leave(r.ctx, "I inspected the files without changing them."); err != nil || !departed {
		t.Fatalf("contribution could not close after reading the instruction: %t %v", departed, err)
	}
}

type failedPublicStore struct{ *memoryRoom }

var errPublicWrite = errors.New("public message transaction failed")

func (s failedPublicStore) AppendTeamMessage(ctx context.Context, message channel.Message) (channel.Message, error) {
	if message.From == "Hana" {
		return message, errPublicWrite
	}
	return s.memoryRoom.AppendTeamMessage(ctx, message)
}

func TestFailedPublicWriteCancelsAndJoinsRunningPeers(t *testing.T) {
	ready := make(chan struct{})
	var joined, closed atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := Run(ctx, Options{Store: failedPublicStore{&memoryRoom{}}, Emit: quiet, MaxModelCalls: 10, MaxSeconds: 5,
		Execute: func(ctx context.Context, member identity.Member, _ string, peer *Peer) (run.Result, error) {
			defer closed.Add(1)
			if joined.Add(1) == 2 {
				close(ready)
			}
			<-ready
			if member.ID == "Hana" {
				err := peer.Emit(runrecord.Event{Kind: "progress", Text: "A result that must be recorded"})
				return run.Result{}, err
			}
			<-ctx.Done()
			return run.Result{}, ctx.Err()
		}}, "Complete the requested contribution")
	if !errors.Is(err, errPublicWrite) || closed.Load() != 2 {
		t.Fatalf("write failure was hidden or workers retained: closed=%d error=%v", closed.Load(), err)
	}
}

func TestFinalPeerCleanupFailureIsReturnedAfterJoining(t *testing.T) {
	want := errors.New("final private session flush failed")
	goodbye := make(chan struct{})
	result, err := Run(context.Background(), Options{Store: &memoryRoom{}, Emit: quiet, MaxModelCalls: 10, MaxSeconds: 5,
		Execute: func(ctx context.Context, member identity.Member, _ string, peer *Peer) (run.Result, error) {
			if member.ID == "Ren" {
				if _, err := peer.Inbox(ctx); err != nil {
					return run.Result{}, err
				}
				peer.BeginDecision()
				left, err := peer.Leave(ctx, "I finished the requested contribution.")
				close(goodbye)
				return run.Result{Departed: left}, err
			}
			<-goodbye
			if _, err := peer.Inbox(ctx); err != nil {
				return run.Result{}, err
			}
			peer.BeginDecision()
			if _, err := peer.AfterAnswer(ctx, "Completed changes; checks described above."); err != nil {
				return run.Result{}, err
			}
			return run.Result{Text: "Completed changes; checks described above."}, want
		}}, "Complete the request")
	if !errors.Is(err, want) || result.Status != runrecord.Stopped {
		t.Fatalf("cleanup failure hidden: %+v %v", result, err)
	}
}
