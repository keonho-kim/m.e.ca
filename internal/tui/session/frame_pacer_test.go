// Purpose: Preserve fast streaming cadence while bounding slow automatic paints and prompt input echo.
// Pattern: Deterministic frame-deadline behavior tests.
// Usage: Session tests supply a clock and measured frame durations without timing sleeps.
// Related: internal/tui/session/frame_pacer.go, internal/tui/session/event_loop.go.
package session

import (
	"testing"
	"time"
)

func TestFramePacerDeadlines(t *testing.T) {
	start := time.Unix(0, 0)
	var p framePacer
	p.painted(start, start.Add(time.Millisecond), false)
	if got := p.delay(p.finished, false); got != frameInterval {
		t.Fatalf("fast frame cadence changed: %v", got)
	}
	if got := p.delay(p.finished, true); got != frameInterval-time.Millisecond {
		t.Fatalf("fast input frames exceeded their budget: %v", got)
	}
	p.painted(start, start.Add(12*time.Millisecond), false)
	if got := p.delay(p.finished, false); got != 36*time.Millisecond {
		t.Fatalf("slow automatic frame consumed its event budget: %v", got)
	}
	if got := p.delay(p.finished, true); got != 0 {
		t.Fatalf("input waited for automatic frame backoff: %v", got)
	}
	p.painted(start, start.Add(12*time.Millisecond), true)
	if got := p.delay(p.finished, true); got != frameInterval-12*time.Millisecond {
		t.Fatalf("consecutive input frames exceeded their budget: %v", got)
	}
	p.painted(start, start.Add(time.Second), false)
	if got := p.delay(p.finished, false); got != 250*time.Millisecond {
		t.Fatalf("automatic backoff exceeded display limit: %v", got)
	}
	if got := p.delay(p.finished, true); got != 0 {
		t.Fatalf("ready input was delayed after an expensive frame: %v", got)
	}
	p.painted(start, start.Add(time.Millisecond), false)
	if got := p.delay(p.finished, false); got != frameInterval {
		t.Fatalf("frame pacing did not recover after fast output: %v", got)
	}
}
