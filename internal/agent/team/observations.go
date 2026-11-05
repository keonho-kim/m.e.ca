// Purpose: Share compact durable execution facts without copying private output or reasoning.
// Pattern: Ordered publication derived from committed operation records.
// Usage: Peer.Emit commits an original reference and its compact fact before presentation.
// Related: internal/agent/team/publication.go, internal/agent/team/contribution/state.go.
package team

import (
	"errors"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (p *Peer) observe(event runrecord.Event) (observed bool, resultErr error) {
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
	defer func() { resultErr = errors.Join(resultErr, r.endPublication()) }()
	if op.Process != nil {
		r.mu.Lock()
		live := r.active[p.ID]
		previous := commandStatus{}
		if live != nil {
			previous = live.statuses[op.Process.CommandID]
		}
		r.mu.Unlock()
		running := previous.sequence != 0
		initial := op.Name == "execute_command" && event.Kind != "command_finished"
		if running && previous.status == op.Process.Status || !running && !initial {
			return false, nil
		}
	}
	value, err := r.storePublic(message)
	if err != nil {
		return false, err
	}
	effect := publicEffect{command: op.Process}
	if hasEvidence {
		effect.evidence = &evidence
	}
	_, err = r.acceptPublic(value, effect)
	return true, err
}
