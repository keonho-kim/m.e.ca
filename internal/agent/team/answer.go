// Purpose: Keep a peer's native run alive while idle and elect one final reporter from actual unread work.
// Pattern: Event-driven answer boundary with predicate-checked wakeups.
// Usage: The team run supplies AfterAnswer; ordinary single runs have no callback.
// Related: internal/agent/run/loop.go, internal/agent/team/worker.go.
package team

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

// AfterAnswer retains the current execution's context and bounded encoding cache.
// A buffered notification is only a hint: consumed messages never start another model call.
func (p *Peer) AfterAnswer(ctx context.Context, text string) (bool, error) {
	r := p.room
	announced := false
	for {
		if err := r.admitUser(); err != nil {
			return false, err
		}
		r.memberMu.Lock()
		r.logMu.Lock()
		r.mu.Lock()
		through, count, remaining := r.latestPublic, r.countLocked(), r.remainingLocked()
		live := r.active[p.ID]
		r.mu.Unlock()
		pending, err := r.options.Store.TeamPending(ctx, p.Cursor, through, p.ID)
		if err == nil && count == 1 && !pending {
			_, err = r.appendPublic(channel.Message{From: p.ID, Text: text, FromStream: true, Final: true})
			if err == nil {
				r.mu.Lock()
				r.result.Text, r.result.Agent = text, p.ID
				r.finished = true
				r.mu.Unlock()
			}
			r.logMu.Unlock()
			r.memberMu.Unlock()
			r.notify()
			return false, err
		}
		if err == nil && !announced && (pending || remaining > 1) {
			_, err = r.appendPublic(channel.Message{From: p.ID, Text: text, FromStream: true})
			announced = true
		}
		r.logMu.Unlock()
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
