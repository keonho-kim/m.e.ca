// Purpose: Expose one model-facing peer identity, receive cursor and shared-room admission facts.
// Pattern: Peer-owned cursor with room-owned synchronized state.
// Usage: Private peer composition supplies these methods to the ordinary model loop.
// Related: internal/agent/team/inbox.go, internal/agent/team/membership.go.
package team

import "time"

type Peer struct {
	ID                   string
	Cursor               int64
	UserSequence         int64
	DecisionCursor       int64
	DecisionUserSequence int64
	receivedView         decisionSnapshot
	decisionView         decisionSnapshot
	contextView          *workProjection
	contextMask          uint16
	contextText          string
	idleDirective        string
	room                 *Room
	// Only commands awaiting their first terminal observation remain here.
	// Publication owns this map under room.logMu; completed archives need no entry.
	running map[string]struct{}
}

func (p *Peer) Deadline() time.Time { return p.room.deadline }
func (p *Peer) AdmitModel() bool {
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.modelCalls >= r.options.MaxModelCalls || !time.Now().Before(r.deadline) || r.ctx.Err() != nil {
		return false
	}
	r.modelCalls++
	return true
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
