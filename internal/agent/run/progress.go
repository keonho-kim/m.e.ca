// Purpose: Detect repeated file states and offer bounded recovery before stopping a stalled execution.
// Pattern: Progress state transitions with durable recovery notices.
// Usage: action observes content changes; Run handles the configured no-progress threshold.
// Related: internal/agent/run/run.go, internal/agent/run/action.go, internal/tools/local/mutations.go.
package run

import (
	"context"
	"fmt"
)

func (s *execution) recoverProgress(ctx context.Context) error {
	if s.checkpoint.Recoveries >= 2 {
		return fmt.Errorf("execution stalled after two recovery rounds without sufficient new evidence; last observation: %s", excerpt(s.checkpoint.Observation, 512))
	}
	s.checkpoint.Recoveries++
	s.checkpoint.NoProgress = 0
	s.checkpoint.Observation = fmt.Sprintf("Recovery %d/2: reuse observed results and choose a different action. Read only a missing range or changed state. Inspect pending checks and actual failures. If blocked, explain the blocker in your answer. Previous observation: %s", s.checkpoint.Recoveries, excerpt(s.checkpoint.Observation, 1024))
	if err := s.record(ctx, Event{Kind: "recovery", Text: s.checkpoint.Observation}); err != nil {
		return err
	}
	return s.persist(ctx)
}
