// Purpose: Register atomic public contributions and expose current work to private model decisions.
// Pattern: Room-owned state with serialized public publication.
// Usage: Team-only tools and executor admission call Peer work methods.
// Related: internal/agent/team/contribution/state.go, internal/commands/teamrequest/peer.go.
package team

import (
	"errors"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/contribution"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (p *Peer) SetWork(summary string, paths []string, contract string, replyTo int64) (result contribution.Scope, resultErr error) {
	r := p.room
	if err := r.admitUser(); err != nil {
		return contribution.Scope{}, err
	}
	note, err := p.note(r.ctx, replyTo, nil)
	if err != nil {
		return contribution.Scope{}, err
	}
	if r.options.Redact != nil {
		summary, contract = r.options.Redact(summary), r.options.Redact(contract)
	}
	requested := contribution.Describe(contribution.Scope{Member: p.ID, Summary: summary, Paths: paths, Contract: contract, ReplyTo: replyTo})
	r.logMu.Lock()
	defer func() { resultErr = errors.Join(resultErr, r.endPublication()) }()
	if err := r.ctx.Err(); err != nil {
		return contribution.Scope{}, err
	}
	r.mu.Lock()
	live := r.active[p.ID]
	if live == nil || live.closed || live.leaving {
		r.mu.Unlock()
		return contribution.Scope{}, faults.Errorf(diagcatalog.TeamThisTeamMemberIsNoLongerActive)
	}
	if err := p.checkUserDecisionLocked(); err != nil {
		r.mu.Unlock()
		return contribution.Scope{}, err
	}
	candidate, err := r.work.Prepare(requested)
	if err == nil && r.work.Matches(candidate) {
		value := r.work.Scope(p.ID)
		r.mu.Unlock()
		return value, nil
	}
	if err != nil {
		r.mu.Unlock()
		return contribution.Scope{}, &faults.Failure{Code: diagcatalog.TeamWorkConflict, Detail: err.Error(), Next: diagcatalog.Text(diagcatalog.TeamReadTheNamedTeammateSWorkAgree)}
	}
	r.mu.Unlock()
	note.Work = &runrecord.TeamWork{Paths: candidate.Paths, Contract: contract, Revision: candidate.Revision}
	value, err := r.storePublic(channel.Message{From: p.ID, Text: summary, Declaration: true, Note: note})
	if err != nil {
		return contribution.Scope{}, err
	}
	if _, err := r.acceptPublic(value, publicEffect{work: &candidate}); err != nil {
		return contribution.Scope{}, err
	}
	r.mu.Lock()
	current := r.work.Scope(p.ID)
	r.mu.Unlock()
	return current, nil
}

// BeginDecision captures the last conversation actually admitted to this model call.
func (p *Peer) BeginDecision() {
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	p.DecisionCursor = p.Cursor
	p.DecisionUserSequence = p.UserSequence
	p.decisionView = p.receivedView
	if live := r.active[p.ID]; live != nil {
		live.reviewed = p.Cursor
		live.waiting = false
	}
	if p.contextView != p.decisionView.view {
		p.contextView, p.contextText = nil, ""
	}
	through := p.Cursor
	for _, live := range r.active {
		if !live.closed && !live.leaving {
			through = min(through, live.reviewed)
		}
	}
	r.work.Prune(through)
}

func (p *Peer) CheckWork(paths []string) error {
	r := p.room
	r.mu.Lock()
	err := r.work.Check(p.ID, paths, p.DecisionCursor)
	r.mu.Unlock()
	if err == nil {
		return nil
	}
	return &faults.Failure{Code: diagcatalog.TeamWorkConflict, Detail: diagcatalog.Text(diagcatalog.TeamNothingExecutedS, err.Error()), Next: diagcatalog.Text(diagcatalog.TeamReadTheCurrentTeamWorkAndActual)}
}

func (p *Peer) HasWorkPaths(paths []string) bool {
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.work.HasPaths(p.ID, paths)
}
