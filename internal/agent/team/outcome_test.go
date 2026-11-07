// Purpose: Preserve engine-owned terminal causes when team execution crosses application boundaries.
// Pattern: Deterministic terminal-state regression tests.
// Usage: go test ./internal/agent/team -run TestTeamTerminalOutcome.
// Related: internal/agent/team/room.go, internal/commands/server/execution.go.
package team

import (
	"context"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/work"
)

func TestTeamTerminalOutcome(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cause    error
		deadline time.Time
		want     work.Status
	}{
		{"completed", nil, time.Time{}, work.Completed},
		{"cancelled", context.Canceled, time.Time{}, work.Interrupted},
		{"parent deadline", context.DeadlineExceeded, time.Now().Add(time.Hour), work.Interrupted},
		{"team budget", context.DeadlineExceeded, time.Now().Add(-time.Hour), work.Exhausted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var saved record.Event
			r := &Room{ctx: t.Context(), runID: "run_outcome", deadline: tc.deadline, result: Result{Text: "answer", Started: time.Now()}, options: Options{
				Record: func(_ context.Context, event record.Event) error { saved = event; return nil },
				Emit:   quiet,
			}}
			result, _ := r.complete(tc.cause)
			if result.End.Status != tc.want || saved.Checkpoint.End.Status != result.End.Status {
				t.Fatalf("returned=%+v persisted=%+v", result.End, saved.Checkpoint.End)
			}
		})
	}
}
