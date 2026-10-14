// Purpose: Detect repeated file states and offer bounded recovery before stopping a stalled task.
// Pattern: Progress state transitions with durable recovery notices.
// Usage: action observes content changes; Run handles the configured no-progress threshold.
// Related: internal/agent/run/run.go, internal/agent/run/action.go, internal/tools/local/mutations.go.
package run

import (
	"context"
	"fmt"
)

func (s *task) recoverProgress(ctx context.Context) error {
	if s.checkpoint.Recoveries >= 2 {
		return fmt.Errorf("task stalled after two recovery rounds without sufficient new evidence")
	}
	s.checkpoint.Recoveries++
	s.checkpoint.NoProgress = 0
	s.checkpoint.Observation = fmt.Sprintf("Recovery %d/2: progress stalled. Re-read current state, narrow the target or choose a different hypothesis. Compare remaining verification failures and prior approaches. Any rollback is a new scoped, approved change, never automatic. Do not replay an executed operation. If blocked, use WAIT. Previous observation: %s", s.checkpoint.Recoveries, excerpt(s.checkpoint.Observation, 4096))
	if err := s.record(ctx, Event{Kind: "recovery", Text: s.checkpoint.Observation}); err != nil {
		return err
	}
	return s.persist(ctx)
}
