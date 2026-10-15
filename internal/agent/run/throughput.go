// Purpose: Calculate average output throughput from received token usage and monotonic elapsed time.
// Pattern: Pure function.
// Usage: Generation attaches this transient value to usage events; it is not persisted.
// Related: internal/agent/run/usage.go, internal/agent/run/events.go.
package run

import "time"

func outputRate(tokens int64, elapsed time.Duration) *float64 {
	if tokens < 0 || elapsed <= 0 {
		return nil
	}
	rate := float64(tokens) / elapsed.Seconds()
	return &rate
}
