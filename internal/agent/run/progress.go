// Purpose: Stop stalled work with the recorded outcomes and remaining uncertainty.
// Pattern: Progress state transition.
// Usage: action observes content changes; Run handles the configured no-progress threshold.
// Related: internal/agent/run/run.go, internal/agent/run/action.go, internal/tools/local/mutations.go.
package run

import (
	"context"
)

func (s *execution) recoverProgress(ctx context.Context) error {
	_, err := s.report(ctx, "Work stopped making progress; inspect the recorded outcomes before continuing the remaining work.", true)
	s.terminal = err == nil
	return err
}
