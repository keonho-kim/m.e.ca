// Purpose: Describe the timing source and live output speed of one model attempt.
// Pattern: Measurement values independent of terminal rendering.
// Usage: The runner records attempt metadata and sends transient snapshots to TUI subscribers.
// Related: internal/agent/run/attempt_metrics.go, internal/agent/record/events.go.
package record

import (
	"time"

	"github.com/keonho-kim/meca/internal/agent/inference"
)

type OutputSpeed struct {
	Source string `json:"source"`
	Throughput
}

type ModelActivity struct {
	Role                 inference.Role
	Started, FirstOutput time.Time
	Finished             bool
	Speed                OutputSpeed
}
