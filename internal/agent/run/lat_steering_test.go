// Purpose: Ensure steering cancels and joins a slow advisory review before new work.
// Pattern: Deterministic cancellation test with virtual time.
// Usage: go test -race ./internal/agent/run -run TestLATSteeringCancels.
// Related: internal/agent/run/lat.go, internal/agent/run/command_generation.go.
package run

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestLATSteeringCancelsAndJoinsBlockedReview(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, model, _, _ := latFixture(nativeCall("inspect_fixture", map[string]any{"path": "a"}), "Old draft", "New answer")
		var pending atomic.Bool
		joined := make(chan struct{})
		r.PollSteering = func() []string {
			if pending.Swap(false) {
				return []string{"A different request"}
			}
			return nil
		}
		model.review = func(ctx context.Context, _ modelapi.Request, _ func(modelapi.Event) error) error {
			pending.Store(true)
			<-ctx.Done()
			close(joined)
			return ctx.Err()
		}
		result, err := r.Run(context.Background(), "Inspect a", approve, emit)
		if err != nil || result.Text != "New answer" || len(model.reviews) != 1 {
			t.Fatalf("steering waited for stale review: %+v %v", result, err)
		}
		select {
		case <-joined:
		default:
			t.Fatal("review worker outlived its steering boundary")
		}
	})
}
