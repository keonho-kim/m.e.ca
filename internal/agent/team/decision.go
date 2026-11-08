// Purpose: Reject team effects based on user instructions not yet reviewed by the model.
// Pattern: Shared decision-version policy at execution boundaries.
// Usage: Local effects and native collaboration check the same captured user sequence.
// Related: internal/agent/team/work.go, internal/tools/team/local.go.
package team

import (
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (p *Peer) CheckUserDecision() error {
	if err := p.room.admitUser(); err != nil {
		return err
	}
	p.room.mu.Lock()
	defer p.room.mu.Unlock()
	return p.checkUserDecisionLocked()
}

func (p *Peer) checkUserDecisionLocked() error {
	if p.DecisionUserSequence < p.room.latestUser {
		return faults.NewInstructionChanged()
	}
	return nil
}

// UserChanges returns one shared generation signal without per-operation watchers.
// The room closes it on new user input; already-running effects are unaffected.
func (p *Peer) UserChanges() (<-chan struct{}, error) {
	if err := p.room.admitUser(); err != nil {
		return nil, err
	}
	r := p.room
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := p.checkUserDecisionLocked(); err != nil {
		return nil, err
	}
	if r.userChanged == nil {
		r.userChanged = make(chan struct{})
	}
	return r.userChanged, nil
}
