// Purpose: Keep a peer's native run alive while idle and elect one final reporter from actual unread work.
// Pattern: Event-driven answer boundary with predicate-checked wakeups.
// Usage: The team run supplies AfterAnswer; ordinary single runs have no callback.
// Related: internal/agent/run/loop.go, internal/agent/team/worker.go.
package team

import (
	"context"
	"errors"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

// AfterAnswer retains the current execution's context and bounded encoding cache.
// A buffered notification is only a hint: consumed messages never start another model call.
func (p *Peer) AfterAnswer(ctx context.Context, text string) (bool, error) {
	r := p.room
	p.idleDirective = ""
	announced := false
	for {
		if err := r.admitUser(); err != nil {
			return false, err
		}
		r.memberMu.Lock()
		r.logMu.Lock()
		r.mu.Lock()
		count, remaining := r.countLocked(), r.remainingLocked()
		live := r.active[p.ID]
		pending := live.attention > p.DecisionCursor
		live.waiting = !pending
		resume, idleErr := r.resolveIdleLocked()
		if resume {
			live.waiting = false
			pending = true
			p.idleDirective = "All active teammates have finished responding and no command is running. Select a concrete remaining contribution and act, or use leave_team if your contribution is complete. Explain an actual blocker instead of asking another idle teammate to decide."
		}
		r.mu.Unlock()
		if idleErr != nil {
			idleErr = errors.Join(idleErr, r.endPublication())
			r.memberMu.Unlock()
			r.fail(idleErr)
			return false, idleErr
		}
		var err error
		if count == 1 && !pending {
			_, err = r.appendPublic(p.answerMessage(text, true))
			if err == nil {
				r.mu.Lock()
				r.result.Text, r.result.Agent = text, p.ID
				r.finished = true
				r.mu.Unlock()
			}
			err = errors.Join(err, r.endPublication())
			r.memberMu.Unlock()
			r.notify()
			return false, err
		}
		if !announced && (pending || remaining > 1) {
			_, err = r.appendPublic(p.answerMessage(text, false))
			announced = true
		}
		err = errors.Join(err, r.endPublication())
		r.memberMu.Unlock()
		if err != nil || pending {
			return pending, err
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-live.wake:
		}
	}
}

func (p *Peer) answerMessage(text string, final bool) channel.Message {
	p.room.mu.Lock()
	basedOn := p.DecisionCursor
	p.room.mu.Unlock()
	return channel.Message{From: p.ID, Text: text, FromStream: true, Final: final, Note: &runrecord.TeamNote{BasedOn: basedOn}}
}
