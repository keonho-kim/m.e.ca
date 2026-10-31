// Purpose: Reserve event-loop time for input when automatic terminal frames become expensive.
// Pattern: Pure frame scheduling policy with caller-supplied times.
// Usage: The terminal event loop records render cost and asks when each pending frame may run.
// Related: internal/tui/session/event_loop.go, internal/tui/session/frame_pacer_test.go.
package session

import "time"

type framePacer struct {
	started, finished time.Time
	cost              time.Duration
	inputFrame        bool
}

func (p *framePacer) painted(started, finished time.Time, input bool) {
	p.started, p.finished, p.cost = started, finished, finished.Sub(started)
	p.inputFrame = input
}

func (p framePacer) delay(now time.Time, input bool) time.Duration {
	// Automatic paint leaves up to three render costs for processing events.
	// Keep fast streams unchanged and limit automatic backoff to 250ms.
	due := p.finished.Add(max(frameInterval, min(3*p.cost, 250*time.Millisecond)))
	if input {
		// Input can use the next 60Hz slot measured from the previous frame's
		// start, without waiting for an automatic backoff after that frame ends.
		due = p.started.Add(frameInterval)
		if !p.inputFrame && 3*p.cost > frameInterval {
			// Input arriving during a slow automatic paint gets one immediate
			// echo. Consecutive input frames still respect the 60Hz budget.
			due = p.finished
		}
	}
	return max(0, due.Sub(now))
}
