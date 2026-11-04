// Purpose: Calculate weighted output throughput from actual counts and their measured intervals.
// Pattern: Small immutable value reducer.
// Usage: Single/team displays and final summaries share Throughput without retaining usage payloads.
// Related: internal/agent/record/events.go, internal/tui/session/tokens.go, internal/tui/chat/team/state.go.
package record

import "time"

type Throughput struct {
	Output   int64
	Duration time.Duration
}

// MeasuredThroughput selects only reported output with a matching observation interval.
// Durable history supplies milliseconds; live events preserve monotonic timer precision.
func MeasuredThroughput(event Event) Throughput {
	if event.Usage == nil || event.Usage.Output < 0 || event.MeasuredDuration < 0 {
		return Throughput{}
	}
	duration := event.MeasuredDuration
	if duration == 0 && event.Generation != nil {
		duration = time.Duration(event.Generation.MeasuredMillis) * time.Millisecond
	}
	if duration <= 0 {
		return Throughput{}
	}
	return Throughput{Output: event.Usage.Output, Duration: duration}
}

func (t Throughput) Add(other Throughput) Throughput {
	return Throughput{Output: t.Output + other.Output, Duration: t.Duration + other.Duration}
}

func (t Throughput) Rate() *float64 {
	if t.Output < 0 || t.Duration <= 0 {
		return nil
	}
	rate := float64(t.Output) / t.Duration.Seconds()
	return &rate
}
