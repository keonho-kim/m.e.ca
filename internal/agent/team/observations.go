// Purpose: Share compact durable execution facts without copying private output or reasoning.
// Pattern: Ordered publication derived from committed operation records.
// Usage: Peer.Emit commits an original reference and its compact fact before presentation.
// Related: internal/agent/team/publication.go, internal/agent/team/contribution/state.go.
package team

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (p *Peer) observe(event runrecord.Event) (bool, error) {
	op := event.Operation
	if op == nil || len(op.Changes) == 0 && op.Process == nil && (!op.Started || op.Failure == nil) {
		return false, nil
	}
	r := p.room
	// Projection and output hashing borrow the immutable event only for this call.
	// Keep their linear work outside the room's ordered append boundary.
	message := operationObservation(p.ID, r.options.ProjectRoot, event)
	evidence, hasEvidence := operationProgress(event)
	r.logMu.Lock()
	defer r.logMu.Unlock()
	if op.Process != nil {
		_, running := p.running[op.Process.CommandID]
		initial := op.Name == "execute_command" && event.Kind != "command_finished"
		if running && op.Process.Status == "running" || !running && !initial {
			return false, nil
		}
	}
	value, err := r.storePublic(message)
	if err != nil {
		return false, err
	}
	if process := op.Process; process != nil {
		if process.Status == "running" {
			if p.running == nil {
				p.running = make(map[string]struct{})
			}
			p.running[process.CommandID] = struct{}{}
		} else {
			delete(p.running, process.CommandID)
			if len(p.running) == 0 {
				p.running = nil
			}
		}
	}
	r.mu.Lock()
	if live := r.active[p.ID]; live != nil {
		live.commands = len(p.running)
	}
	r.mu.Unlock()
	effect := publicEffect{}
	if hasEvidence {
		effect.evidence = &evidence
	}
	_, err = r.acceptPublic(value, effect)
	return true, err
}
