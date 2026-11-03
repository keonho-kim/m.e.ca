// Purpose: Expose one model-facing peer identity, receive cursor and shared-room admission facts.
// Pattern: Peer-owned cursor with room-owned synchronized state.
// Usage: Private peer composition supplies these methods to the ordinary model loop.
// Related: internal/agent/team/inbox.go, internal/agent/team/membership.go.
package team

import (
	"slices"
	"strings"
	"time"
)

type Peer struct {
	ID           string
	Cursor       int64
	UserSequence int64
	room         *Room
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
func (p *Peer) Names() string {
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make([]string, 0, len(r.active))
	for id, live := range r.active {
		if !live.closed && !live.leaving {
			names = append(names, id)
		}
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}
func (p *Peer) HasNewUserInstruction() bool {
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	return p.UserSequence < r.latestUser
}
