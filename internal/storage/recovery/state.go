// Purpose: Mark unfinished run states interrupted without replaying operations.
// Pattern: Recovery state transitions.
// Usage: Restore invokes these state recovery steps.
// Related: internal/storage/recovery/recovery.go.
package recovery

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/run"
)

func recoverRun(ctx context.Context, s Store, session string) error {
	cp, err := s.LatestRun(ctx, session)
	if err != nil || cp == nil {
		return err
	}
	if cp.Status != run.Running {
		return nil
	}
	cp.Status = run.Interrupted
	return s.RecordEvent(ctx, session, run.Event{Kind: "run_checkpoint", Text: "Run interrupted during " + cp.Phase + "; drafts and side effects were not replayed", Checkpoint: cp})
}
