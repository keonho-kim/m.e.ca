// Purpose: Resolve collective idle states once per actual progress boundary without polling inference.
// Pattern: Room-owned event-driven liveness guard.
// Usage: Answer boundaries invoke this under the room lock after marking the current peer idle.
// Related: internal/agent/team/answer.go, internal/agent/team/observations.go.
package team

import "fmt"

func (r *Room) resolveIdleLocked() (bool, error) {
	if r.remainingLocked() < 2 {
		return false, nil
	}
	for _, live := range r.active {
		if !live.closed && !live.leaving && (!live.waiting || live.commands > 0 || live.attention > live.reviewed) {
			return false, nil
		}
	}
	if r.idleAttempted && r.idleEpoch == r.progress {
		return false, fmt.Errorf("team stalled: all active teammates are waiting, no command is running, and reconsideration produced no progress")
	}
	r.idleAttempted, r.idleEpoch = true, r.progress
	return true, nil
}
