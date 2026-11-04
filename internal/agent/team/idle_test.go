// Purpose: Detect collective waiting without polling models or interrupting active commands.
// Pattern: Deterministic liveness state tests.
// Usage: Team answer boundaries use progress-scoped reconsideration and explicit stalled errors.
// Related: internal/agent/team/idle.go, internal/agent/team/answer.go.
package team

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"strings"
	"testing"
	"time"
)

func TestAllIdleReconsidersOnceThenReportsNoProgress(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	r.active["Ren"].waiting = true
	again, err := p.AfterAnswer(r.ctx, "Waiting for Ren to choose.")
	if err != nil || !again || p.idleDirective == "" {
		t.Fatal("collective idle did not resume once", again, err)
	}
	_, err = p.AfterAnswer(r.ctx, "I am still waiting.")
	if err == nil || !strings.Contains(err.Error(), "team stalled") || r.ctx.Err() == nil {
		t.Fatal("unchanged idle waited indefinitely", err)
	}
}

func TestCollectiveIdleStopsAndJoinsTheActualRoom(t *testing.T) {
	execute := func(ctx context.Context, _ identity.Member, _ string, p *Peer) (run.Result, error) {
		for {
			if _, err := p.Inbox(ctx); err != nil {
				return run.Result{}, err
			}
			p.BeginDecision()
			again, err := p.AfterAnswer(ctx, "Waiting for the other teammate to choose.")
			if err != nil || !again {
				return run.Result{Text: "Waiting"}, err
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := Run(ctx, Options{Store: &memoryRoom{}, Execute: execute, Emit: quiet, MaxModelCalls: 100, MaxSeconds: 10}, "Implement the request")
	if err == nil || !strings.Contains(err.Error(), "team stalled") || ctx.Err() != nil {
		t.Fatal("collective idle waited for the global deadline", err)
	}
}

func TestIdleGuardWaitsForActualWorkAndResetsAfterProgress(t *testing.T) {
	r := testRoom(t)
	for _, live := range r.active {
		live.waiting = true
	}
	r.active["Ren"].commands = 1
	if resume, err := r.resolveIdleLocked(); resume || err != nil {
		t.Fatal("running command was treated as idle")
	}
	r.active["Ren"].commands = 0
	if resume, err := r.resolveIdleLocked(); !resume || err != nil {
		t.Fatal("no reconsideration")
	}
	r.progress++
	if resume, err := r.resolveIdleLocked(); !resume || err != nil {
		t.Fatal("actual progress did not reset the guard")
	}
	r.active["Hana"].attention = 10
	if resume, err := r.resolveIdleLocked(); resume || err != nil {
		t.Fatal("unread conversation was treated as deadlock")
	}
}
