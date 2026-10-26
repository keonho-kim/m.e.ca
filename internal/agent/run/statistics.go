// Purpose: Aggregate one request's measured token usage and generation time independently of session totals.
// Pattern: Request-owned measurements.
// Usage: Generation records each attempt once; the final checkpoint publishes a frozen snapshot.
// Related: internal/agent/run/usage.go, internal/agent/run/journal.go, internal/agent/record/events.go.
package run

import (
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

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
	*s.usage = s.usage.Add(*usage)
	s.generation += max(0, elapsed)
}

func (s runStatistics) snapshot(now time.Time) runrecord.Statistics {
	result := runrecord.Statistics{ElapsedMillis: max(0, now.Sub(s.started).Milliseconds()), GenerationMillis: s.generation.Milliseconds()}
	if s.usage != nil {
		usage := s.usage.Clone()
		result.Usage = &usage
	}
	return result
}
