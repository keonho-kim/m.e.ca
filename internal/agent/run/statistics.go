// Purpose: Aggregate one request's measured token usage and generation time independently of session totals.
// Pattern: Request-owned measurements.
// Usage: Generation records each attempt once; the final checkpoint publishes a frozen snapshot.
// Related: internal/agent/run/usage.go, internal/agent/run/journal.go, internal/agent/run/events.go.
package run

import (
	"fmt"
	"time"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

type Statistics struct {
	ElapsedMillis    int64                `json:"elapsed_millis"`
	GenerationMillis int64                `json:"generation_millis"`
	Usage            *modelapi.TokenUsage `json:"usage,omitempty"`
}

func (s Statistics) check() error {
	if s.ElapsedMillis < 0 || s.GenerationMillis < 0 || s.GenerationMillis > s.ElapsedMillis {
		return fmt.Errorf("invalid request measurement duration")
	}
	if s.Usage == nil {
		if s.GenerationMillis != 0 {
			return fmt.Errorf("generation measurement requires reported token usage")
		}
	} else if s.Usage.Input < 0 || s.Usage.Output < 0 {
		return fmt.Errorf("invalid request token totals")
	}
	return nil
}

type runStatistics struct {
	started    time.Time
	generation time.Duration
	usage      *modelapi.TokenUsage
}

func (s *runStatistics) add(usage *modelapi.TokenUsage, elapsed time.Duration) {
	if usage == nil {
		return
	}
	if s.usage == nil {
		s.usage = &modelapi.TokenUsage{}
	}
	s.usage.Input += usage.Input
	s.usage.Output += usage.Output
	s.generation += max(0, elapsed)
}

func (s runStatistics) snapshot(now time.Time) Statistics {
	result := Statistics{ElapsedMillis: max(0, now.Sub(s.started).Milliseconds()), GenerationMillis: s.generation.Milliseconds()}
	if s.usage != nil {
		usage := *s.usage
		result.Usage = &usage
	}
	return result
}
