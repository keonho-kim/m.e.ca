// Purpose: Detect repeated file states and offer bounded recovery before stopping a stalled execution.
// Pattern: Progress state transitions with durable recovery notices.
// Usage: action observes content changes; Run handles the configured no-progress threshold.
// Related: internal/agent/run/run.go, internal/agent/run/action.go, internal/tools/local/mutations.go.
package run

import (
	"context"
)

func (s *execution) recoverProgress(ctx context.Context) error {
	finished, err := s.judgeCompletion(ctx, "", "repeated actions or file-state cycle")
	if err != nil {
		return err
	}
	if finished {
		s.terminal = true
	}
	return nil
}
