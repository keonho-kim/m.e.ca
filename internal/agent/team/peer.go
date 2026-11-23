// Purpose: Expose one model-facing peer identity, receive cursor and shared-room admission facts.
// Pattern: Peer-owned cursor with room-owned synchronized state.
// Usage: Private peer composition supplies these methods to the ordinary model loop.
// Related: internal/agent/team/inbox.go, internal/agent/team/membership.go.
package team

import (
	"context"
	"time"
)

type Peer struct {
	presentation         string
	ID                   string
	Cursor               int64
	UserSequence         int64
	DecisionCursor       int64
	DecisionUserSequence int64
	receivedView         decisionSnapshot
	decisionView         decisionSnapshot
	contextView          *workProjection
	contextCovered       []bool
	coverageScratch      []bool
	contextText          string
	idleDirective        string
	room                 *Room
}

func (p *Peer) Deadline() time.Time { return p.room.deadline }
func (p *Peer) AdmitModel(ctx context.Context) (bool, error) {
	r := p.room
	for {
		r.mu.Lock()
		if err := ctx.Err(); err != nil {
			r.mu.Unlock()
			return false, err
		}
		if err := r.ctx.Err(); err != nil {
			r.mu.Unlock()
			return false, err
		}
		if !time.Now().Before(r.deadline) {
			r.mu.Unlock()
			return false, context.DeadlineExceeded
		}
		live := r.active[p.ID]
		firstReservedCall := r.releaseFirstCallLocked(live)
		if firstReservedCall || r.modelCalls < r.options.MaxModelCalls-r.reservedModelCalls {
			r.modelCalls++
			r.mu.Unlock()
			if firstReservedCall {
				r.logInvitationPhase(r.ctx, live.invitationID, "invite_first_model", live.inviteStarted)
			}
			return true, nil
		}
		if r.modelCalls >= r.options.MaxModelCalls {
			r.budgetExhausted = true
			r.mu.Unlock()
			return false, nil
		}
		if r.modelBudgetChanged == nil {
			r.modelBudgetChanged = make(chan struct{})
		}
		changed := r.modelBudgetChanged
		r.mu.Unlock()
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-r.ctx.Done():
			return false, r.ctx.Err()
		case <-changed:
		}
	}
}

// releaseFirstCallLocked returns an invitation's unused model reservation or consumes it at admission.
func (r *Room) releaseFirstCallLocked(live *participant) bool {
	if live == nil || !live.firstCallReserved {
		return false
	}
	live.firstCallReserved = false
	r.returnModelCallLocked()
	return true
}

func (r *Room) returnModelCallLocked() {
	r.reservedModelCalls--
	if r.modelBudgetChanged != nil {
		close(r.modelBudgetChanged)
		r.modelBudgetChanged = nil
	}
}
func (p *Peer) InitialCursor() int64 { return max(0, p.room.firstMessage-1) }

// HasSpoken reports a committed public response in this active lifetime.
// Private reasoning, streamed drafts and restored historical speech do not count.
func (p *Peer) HasSpoken() bool {
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	live := r.active[p.ID]
	return live != nil && !live.closed && !live.leaving && live.spoke
}
