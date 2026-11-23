// Purpose: Admit one independent contribution and join its private peer session.
// Pattern: Reserve under room state, prepare privately, then serialize activation.
// Usage: add_team_member calls AddMember with a stable public work identity.
// Related: internal/agent/team/worker.go, internal/agent/team/work.go, internal/agent/team/invitation_timing.go.
package team

import (
	"context"
	"errors"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"github.com/keonho-kim/meca/internal/agent/team/source"
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type invitationReservation struct {
	id      uint64
	work    source.WorkDomain
	member  string
	inviter string
	started time.Time
}

func (p *Peer) AddMember(ctx context.Context, invitation Invitation) (result map[string]any, resultErr error) {
	started := time.Now()
	r := p.room
	work, reference, err := r.parseWorkDomain(ctx, source.WorkDomain{Domain: invitation.Domain, WorkRef: invitation.WorkRef})
	if err != nil {
		return nil, err
	}
	if err := r.admitUser(); err != nil {
		return nil, err
	}
	ticket, status, err := p.reserveInvitation(ctx, invitation.Member, work, started)
	if ticket == nil {
		return status, err
	}
	r.logInvitationPhase(ctx, ticket.id, "invite_reserve", started)
	defer r.releaseInvitation(ticket)

	if r.options.MemoryPressure != nil {
		pressured, err := r.options.MemoryPressure()
		if err != nil {
			return nil, err
		}
		if pressured {
			r.mu.Lock()
			count := r.countLocked()
			r.mu.Unlock()
			return map[string]any{"status": "capacity_full", "reason": "memory_pressure", "active": count}, nil
		}
	}
	prepare := time.Now()
	var member identity.Member
	if invitation.Member == "" {
		member, err = r.options.Store.CreateTeamAgent(ctx)
	} else {
		member, err = r.options.Store.TeamMember(ctx, invitation.Member)
	}
	r.logInvitationPhase(ctx, ticket.id, "invite_prepare", prepare)
	if err != nil {
		return nil, err
	}
	published := false
	if invitation.Member == "" {
		defer func() {
			if !published {
				flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				resultErr = errors.Join(resultErr, r.options.Store.DiscardTeamAgent(flush, member))
			}
		}()
	}
	result, published, resultErr = p.activateInvitation(ctx, ticket, member, reference, invitation.Message)
	return result, resultErr
}

func (p *Peer) reserveInvitation(ctx context.Context, member string, work source.WorkDomain, started time.Time) (*invitationReservation, map[string]any, error) {
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	if result, err := p.invitationStatusLocked(ctx, member, work); result != nil || err != nil {
		return nil, result, err
	}
	for pending := range r.pendingInvitations {
		if work.Matches(pending.work) || member != "" && pending.member == member {
			return nil, map[string]any{"status": "invitation_pending", "domain": pending.work.Domain, "work_ref": pending.work.WorkRef, "invited_by": pending.inviter}, nil
		}
	}
	count := r.countLocked()
	// Preserve a next decision for current and pending peers before promising a first call.
	if r.options.MaxModelCalls-r.modelCalls <= count+len(r.pendingInvitations) {
		return nil, map[string]any{"status": "capacity_full", "reason": "model_budget", "active": count}, nil
	}
	r.nextInvitationID++
	ticket := &invitationReservation{id: r.nextInvitationID, work: work, member: member, inviter: p.ID, started: started}
	if r.pendingInvitations == nil {
		r.pendingInvitations = make(map[*invitationReservation]struct{})
	}
	r.pendingInvitations[ticket] = struct{}{}
	r.reservedModelCalls++
	return ticket, nil, nil
}

func (r *Room) releaseInvitation(ticket *invitationReservation) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, pending := r.pendingInvitations[ticket]; pending {
		delete(r.pendingInvitations, ticket)
		r.returnModelCallLocked()
	}
}

func (p *Peer) invitationStatusLocked(ctx context.Context, member string, work source.WorkDomain) (map[string]any, error) {
	r := p.room
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := p.checkUserDecisionLocked(); err != nil {
		return nil, err
	}
	if r.finished || r.ctx.Err() != nil {
		return nil, faults.Errorf(catalog.TeamTheCurrentTeamRequestHasEnded)
	}
	if inviter := r.active[p.ID]; inviter == nil || inviter.closed || inviter.leaving {
		return nil, faults.Errorf(catalog.TeamThisTeamMemberIsNoLongerActive)
	}
	if live := r.active[member]; live != nil {
		status := "already_active"
		if live.leaving {
			status = "closing"
		}
		return map[string]any{"status": status, "member_name": member, "active": r.countLocked()}, nil
	}
	if owner := r.domainOwnerLocked(work, ""); owner != "" {
		return map[string]any{"status": "domain_active", "domain": work.Domain, "work_ref": work.WorkRef, "member_name": owner}, nil
	}
	return nil, nil
}

func (r *Room) domainOwnerLocked(work source.WorkDomain, except string) string {
	for name, live := range r.active {
		if name != except && !live.closed && r.work.HoldsDomain(name, live.invited, work) {
			return name
		}
	}
	return ""
}

func (p *Peer) activateInvitation(ctx context.Context, ticket *invitationReservation, member identity.Member, reference channel.Metadata, message string) (result map[string]any, published bool, resultErr error) {
	r := p.room
	defer r.logInvitationPhase(ctx, ticket.id, "invite_activate", time.Now())
	if err := r.admitUser(); err != nil {
		return nil, false, err
	}
	r.memberMu.Lock()
	defer r.memberMu.Unlock()
	r.logMu.Lock()
	defer func() { resultErr = errors.Join(resultErr, r.endPublication()) }()

	r.mu.Lock()
	status, err := p.invitationStatusLocked(ctx, member.ID, ticket.work)
	count, observed := r.countLocked(), p.DecisionCursor
	r.mu.Unlock()
	if status != nil || err != nil {
		return status, false, err
	}
	note := &runrecord.TeamNote{
		ReplyTo: ticket.work.WorkRef, ReplyAuthor: reference.From, Mentions: []string{member.ID}, BasedOn: observed,
	}
	// An append attempt can have committed even when delivery fails.
	_, err = r.appendPublic(channel.Message{From: p.ID, Text: member.ID + " (" + ticket.work.Domain + "): " + message, Note: note})
	if err != nil {
		return nil, true, err
	}
	live := r.reserve(member, ticket)
	if err := r.launch(live); err != nil {
		return nil, true, err
	}
	state := "created"
	if ticket.member != "" {
		state = "restored"
	}
	return map[string]any{"status": state, "member_name": member.ID, "domain": ticket.work.Domain, "work_ref": ticket.work.WorkRef, "active": count + 1}, true, nil
}
