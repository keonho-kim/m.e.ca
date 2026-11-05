// Purpose: Serialize stable peer invitations and departure while keeping historical identities paged in storage.
// Pattern: Membership transitions with repository-owned registration.
// Usage: Team tools invoke Peer methods; start occupies at most eight slots including closing peers.
// Related: internal/agent/team/worker.go, internal/agent/team/types.go.
package team

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"time"
	"unicode/utf8"
)

func (p *Peer) AddMember(ctx context.Context, id, message string) (result map[string]any, resultErr error) {
	r := p.room
	if err := r.admitUser(); err != nil {
		return nil, err
	}
	r.memberMu.Lock()
	defer r.memberMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	count, finished := r.countLocked(), r.finished
	live := r.active[id]
	decisionErr := p.checkUserDecisionLocked()
	r.mu.Unlock()
	if decisionErr != nil {
		return nil, decisionErr
	}
	if finished || r.ctx.Err() != nil {
		return nil, fmt.Errorf("the current team request has ended")
	}
	if live != nil {
		status := "already_active"
		if live.leaving {
			status = "closing"
		}
		return map[string]any{"status": status, "member_name": id, "available": MaxActive - count}, nil
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
	publicationStarted := false
	defer func() {
		if status == "created" && !publicationStarted {
			flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			resultErr = errors.Join(resultErr, r.options.Store.DiscardTeamAgent(flush, member))
		}
	}()
	// Private DB creation may be slow. Public messages and steering remain
	// available throughout preparation; admit them before committing membership.
	if err := r.admitUser(); err != nil {
		return nil, err
	}
	r.logMu.Lock()
	defer func() { resultErr = errors.Join(resultErr, r.endPublication()) }()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	count, finished = r.countLocked(), r.finished
	decisionErr = p.checkUserDecisionLocked()
	r.mu.Unlock()
	if decisionErr != nil {
		return nil, decisionErr
	}
	if finished || r.ctx.Err() != nil {
		return nil, fmt.Errorf("the current team request has ended")
	}
	// A publication attempt may have committed even if delivery then fails.
	// Preserve that registration rather than deleting potentially referenced state.
	publicationStarted = true
	if _, err = r.appendPublic(channel.Message{From: p.ID, Text: member.ID + ": " + message}); err != nil {
		return nil, err
	}
	if err = r.start(member); err != nil {
		return nil, err
	}
	return map[string]any{"status": status, "member_name": member.ID, "active": count + 1, "available": MaxActive - count - 1}, nil
}
func (p *Peer) Leave(ctx context.Context, message string) (depart bool, resultErr error) {
	r := p.room
	if err := r.admitUser(); err != nil {
		return false, err
	}
	r.memberMu.Lock()
	defer r.memberMu.Unlock()
	r.logMu.Lock()
	defer func() { resultErr = errors.Join(resultErr, r.endPublication()) }()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	r.mu.Lock()
	live := r.active[p.ID]
	valid := live != nil && !live.closed && !live.leaving
	decisionErr := p.checkUserDecisionLocked()
	pending := live != nil && r.remainingLocked() > 1 && live.attention > p.DecisionCursor
	r.mu.Unlock()
	if !valid {
		return false, fmt.Errorf("this peer is already leaving")
	}
	if decisionErr != nil {
		return false, decisionErr
	}
	if pending {
		return false, &tool.Failure{Code: "team_feedback_pending", Detail: "Public feedback requiring your attention arrived after the last model decision; nothing closed.", Next: "Review the newly received public conversation in your next decision. Then answer with discuss(reply_to), continue useful work, or choose leave_team again."}
	}
	if _, err := r.appendPublic(channel.Message{From: p.ID, Text: message}); err != nil {
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
		r.work.Release(p.ID)
		r.invalidateWorkLocked()
		return false, nil
	}
	live.leaving = true
	return true, nil
}
func (p *Peer) Members(ctx context.Context, after string) (MemberPage, error) {
	members, err := p.room.options.Store.TeamMembers(ctx, after, 33)
	if err != nil {
		return MemberPage{}, err
	}
	page := MemberPage{Members: make([]MemberStatus, 0, min(32, len(members)))}
	if len(members) > 32 {
		members = members[:32]
		page.Next = members[31].ID
	}
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, member := range members {
		live := r.active[member.ID]
		value := MemberStatus{ID: member.ID, Summary: member.Summary, Active: live != nil && !live.closed && !live.leaving}
		if scope := r.work.Scope(member.ID); scope.Revision > 0 {
			value.Work = &scope
		}
		page.Members = append(page.Members, value)
	}
	return page, nil
}
