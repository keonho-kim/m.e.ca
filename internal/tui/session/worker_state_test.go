// Purpose: Verify cancellation is an idempotent transition and cannot restart compression.
// Pattern: State transition regressions.
// Usage: go test ./internal/tui/session -run TestWorkerState.
// Related: internal/tui/session/worker_state.go, internal/tui/session/activity.go.
package session

import (
	"testing"
)

func TestWorkerStateCancellation(t *testing.T) {
	for _, initial := range []workerState{workerIdle, workerAgent, workerSetup} {
		state := initial
		state.stop()
		state.stop()
		if state.active() != initial.active() || state.agent() != initial.agent() || state.stopping() != initial.active() {
			t.Fatalf("invalid cancellation transition: %v -> %v", initial, state)
		}
	}
	a := screenFixture(t)
	a.work.state = workerAgent
	a.startCompression()
	a.work.state.stop()
	a.stopCompression()
	a.startCompression()
	if a.work.indicatorTimer != nil || a.work.indicatorTicks != nil {
		t.Fatal("late phase event restarted a cancelled timer")
	}
}
