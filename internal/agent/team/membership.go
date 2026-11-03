// Purpose: Serialize stable peer invitations and departure while keeping historical identities paged in storage.
// Pattern: Membership transitions with repository-owned registration.
// Usage: Team tools invoke Peer methods; start occupies at most eight slots including closing peers.
// Related: internal/agent/team/worker.go, internal/agent/team/types.go.
package team

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"unicode/utf8"
)

func (p *Peer) Invite(ctx context.Context, id, message string) (map[string]any, error) {
	r := p.room
	r.memberMu.Lock()
	defer r.memberMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	count, finished := r.countLocked(), r.finished
	live := r.active[id]
	r.mu.Unlock()
	if finished || r.ctx.Err() != nil {
		return nil, fmt.Errorf("the current team request has ended")
	}
	if live != nil {
		status := "already_active"
		if live.leaving {
			status = "closing"
		}
		return map[string]any{"status": status, "agent_id": id, "available": MaxActive - count}, nil
	}
	if count >= MaxActive {
		return map[string]any{"status": "capacity_full", "active": count, "available": 0, "next": "Continue your contribution or coordinate with active peers; do not repeatedly retry the invitation."}, nil
	}
	var member identity.Member
	var err error
	status := "restored"
	if id == "" {
		member, err = r.options.Store.CreateTeamAgent(ctx)
		status = "created"
	} else {
		member, err = r.options.Store.TeamMember(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	if _, err = r.publish(channel.Message{From: p.ID, Text: member.ID + ": " + message}); err != nil {
		return nil, err
	}
	if err = r.start(member); err != nil {
		return nil, err
	}
	return map[string]any{"status": status, "agent_id": member.ID, "active": count + 1, "available": MaxActive - count - 1}, nil
}
func (p *Peer) Goodbye(ctx context.Context, message string) (bool, error) {
	r := p.room
	if err := r.admitUser(); err != nil {
		return false, err
	}
	r.memberMu.Lock()
	defer r.memberMu.Unlock()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	r.mu.Lock()
	live := r.active[p.ID]
	valid := live != nil && !live.closed && !live.leaving
	stale := p.UserSequence < r.latestUser
	r.mu.Unlock()
	if !valid {
		return false, fmt.Errorf("this peer is already leaving")
	}
	if stale {
		return false, fmt.Errorf("a new user instruction is waiting; read it and reconsider the contribution before leaving")
	}
	if _, err := r.publish(channel.Message{From: p.ID, Text: message}); err != nil {
		return false, err
	}
	summary := message
	if len(summary) > 2048 {
		summary = summary[:2048]
		for !utf8.ValidString(summary) {
			summary = summary[:len(summary)-1]
		}
		summary += "…"
	}
	if err := r.options.Store.SetTeamSummary(ctx, p.ID, summary); err != nil {
		r.fail(err)
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.remainingLocked() == 1 {
		return false, nil
	}
	live.leaving = true
	return true, nil
}
func (p *Peer) Friends(ctx context.Context, after string) (FriendPage, error) {
	members, err := p.room.options.Store.TeamMembers(ctx, after, 33)
	if err != nil {
		return FriendPage{}, err
	}
	page := FriendPage{Friends: make([]Friend, 0, min(32, len(members)))}
	if len(members) > 32 {
		members = members[:32]
		page.Next = members[31].ID
	}
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, member := range members {
		live := r.active[member.ID]
		page.Friends = append(page.Friends, Friend{ID: member.ID, Summary: member.Summary, Active: live != nil && !live.closed})
	}
	return page, nil
}
