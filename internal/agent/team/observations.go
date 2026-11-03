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
	r.logMu.Lock()
	defer r.logMu.Unlock()
	if op.Process != nil {
		_, running := p.running[op.Process.CommandID]
		initial := op.Name == "execute_command" && event.Kind != "command_finished"
		if running && op.Process.Status == "running" || !running && !initial {
			return false, nil
		}
	}
	message := operationObservation(p.ID, r.options.ProjectRoot, event)
	if _, err := r.appendPublic(message); err != nil {
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
	return true, nil
}
