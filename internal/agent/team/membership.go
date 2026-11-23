// Purpose: Finalize peer departure and page current and historical membership.
// Pattern: Membership transitions with repository-owned registration.
// Usage: Team tools close a peer or inspect the room roster.
// Related: internal/agent/team/worker.go, internal/agent/team/types.go.
package team

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

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
		return false, faults.Errorf(diagcatalog.TeamThisPeerIsAlreadyLeaving)
	}
	if decisionErr != nil {
		return false, decisionErr
	}
	if pending {
		return false, &faults.Failure{Code: diagcatalog.TeamFeedbackPending, Detail: diagcatalog.Text(diagcatalog.TeamPublicFeedbackRequiringYourAttentionArrivedAfter), Next: diagcatalog.Text(diagcatalog.TeamReviewTheNewlyReceivedPublicConversationIn)}
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
	r.invalidateWorkLocked()
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
		if value.Active {
			value.Domain, value.WorkRef = live.invited.Domain, live.invited.WorkRef
		}
		if scope := r.work.Scope(member.ID); scope.Revision > 0 {
			value.Work = &scope
		}
		page.Members = append(page.Members, value)
	}
	return page, nil
}
