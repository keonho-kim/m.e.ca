// Purpose: Count progress once after an entire model decision and its ordered tool results.
// Pattern: Deterministic state transition.
// Usage: The run owner resets a round before generation and commits it after the batch is joined.
// Related: internal/agent/run/run.go, internal/agent/run/observation.go.
package run

import "context"

type roundProgress struct{ advanced, cycle bool }

func (s *execution) completeRound(ctx context.Context) error {
	if s.round.advanced {
		s.checkpoint.NoProgress = 0
	} else {
		s.checkpoint.NoProgress++
	}
	if s.round.cycle {
		s.checkpoint.NoProgress = max(s.checkpoint.NoProgress, s.limits.StallIterations)
	}
	return s.persist(ctx)
}
