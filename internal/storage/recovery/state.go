// Purpose: Mark unfinished task and Goal states interrupted without replaying operations.
// Pattern: Recovery state transitions.
// Usage: Restore invokes these state recovery steps.
// Related: internal/storage/recovery/recovery.go.
package recovery

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/run"
)

func recoverTask(ctx context.Context, s Store, session string) error {
	cp, err := s.LatestTask(ctx, session)
	if err != nil || cp == nil {
		return err
	}
	if cp.Status != run.Running {
		return nil
	}
	cp.Status = run.Interrupted
	return s.RecordEvent(ctx, session, run.Event{Kind: "task_checkpoint", Text: "Task interrupted during " + cp.Phase + "; drafts and side effects were not replayed", Checkpoint: cp})
}

func recoverGoal(ctx context.Context, s Store, session string) error {
	state, err := s.LatestGoal(ctx, session)
	if err != nil || state == nil {
		return err
	}
	if state.Status != "running" && state.Status != "pending" {
		return nil
	}
	state.Status = "interrupted"
	for i := range state.Tasks {
		if state.Tasks[i].Status == "running" {
			state.Tasks[i].Status = "interrupted"
		}
	}
	return s.RecordEvent(ctx, session, run.Event{Kind: "goal_state", Goal: state, Internal: true})
}
