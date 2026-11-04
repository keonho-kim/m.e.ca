// Purpose: Wake idle team requests through explicit input notifications and skip empty-history SQL checks.
// Pattern: Virtual-time lifecycle regression tests.
// Usage: Team tests verify input delivery without polling timers or real model requests.
// Related: internal/agent/team/room.go, internal/agent/team/inbox.go, internal/agent/team/answer.go.
package team

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
)

type readCountingRoom struct {
	*memoryRoom
	reads atomic.Int64
}

func (s *readCountingRoom) TeamMessages(ctx context.Context, after, through int64, limit int) ([]channel.Message, error) {
	s.reads.Add(1)
	return s.memoryRoom.TeamMessages(ctx, after, through, limit)
}

func TestTeamInputNotificationDeliversWhileAllWorkersAreIdle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ready := make(chan struct{}, 1)
		store := &memoryRoom{}
		var polls atomic.Int64
		pending := []string{}
		joined, done := make(chan struct{}, 2), make(chan error, 1)
		go func() {
			_, err := Run(ctx, Options{Store: store, Emit: quiet, MaxModelCalls: 100, MaxSeconds: 60, SteeringReady: ready,
				PollSteering: func() []string { polls.Add(1); out := pending; pending = nil; return out },
				Execute: func(ctx context.Context, _ identity.Member, _ string, _ *Peer) (run.Result, error) {
					joined <- struct{}{}
					<-ctx.Done()
					return run.Result{}, ctx.Err()
				},
			}, "Initial request")
			done <- err
		}()
		<-joined
		<-joined
		synctest.Wait()
		time.Sleep(time.Second)
		if polls.Load() != 0 {
			t.Fatal("idle input was polled", polls.Load())
		}
		pending = []string{"A new user instruction"}
		ready <- struct{}{}
		synctest.Wait()
		values, _ := store.TeamMessages(ctx, 0, 0, 10)
		if len(values) != 2 || values[1].Text != "A new user instruction" || polls.Load() != 1 {
			t.Fatal("input notification lost or duplicated the instruction", values, polls.Load())
		}
		close(ready)
		synctest.Wait()
		count := polls.Load()
		time.Sleep(time.Second)
		if polls.Load() != count {
			t.Fatal("closed notification channel caused polling")
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	})
}

func TestAnswerBoundarySkipsSQLWhenNoNewConversationExists(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := testRoom(t)
		store := &readCountingRoom{memoryRoom: r.options.Store.(*memoryRoom)}
		r.options.Store = store
		r.active["Ren"].leaving = true
		ctx, cancel := context.WithCancel(r.ctx)
		done := make(chan error, 1)
		go func() { _, err := (&Peer{ID: "Hana", room: r}).AfterAnswer(ctx, "Done"); done <- err }()
		synctest.Wait()
		if store.reads.Load() != 0 {
			t.Fatal("unchanged conversation queried storage")
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	})
}
