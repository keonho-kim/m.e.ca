// Purpose: Resolve collective idle states once per actual progress boundary without polling inference.
// Pattern: Room-owned event-driven liveness guard.
// Usage: Answer boundaries invoke this under the room lock after marking the current peer idle.
// Related: internal/agent/team/answer.go, internal/agent/team/observations.go.
package team

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (r *Room) resolveIdleLocked() (bool, error) {
	if r.remainingLocked() < 2 {
		return false, nil
	}
	for _, live := range r.active {
		if !live.closed && !live.leaving && (!live.waiting || len(live.statuses) > 0 || live.attention > live.reviewed) {
			return false, nil
		}
	}
	if r.idleAttempted && r.idleEpoch == r.progress {
		return false, faults.Errorf(diagcatalog.TeamTeamStalledAllActiveTeammatesAreWaiting)
	}
	r.idleAttempted, r.idleEpoch = true, r.progress
	return true, nil
}
