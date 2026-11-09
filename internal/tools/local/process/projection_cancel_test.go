// Purpose: Remove canceled memory waiters without blocking later projections or releasing foreign permits.
// Pattern: Deterministic admission lifecycle test.
// Usage: go test -race ./internal/tools/local/process -run TestProjectionCanceledWaiter.
// Related: internal/tools/local/process/projection.go.
package process

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
)

func TestProjectionCanceledWaiter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var budget ProjectionBudget
		release, err := budget.Acquire(t.Context(), 32<<20)
		if err != nil {
			t.Fatal(err)
		}
		defer release()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() { _, err := budget.Acquire(ctx, 1<<20); done <- err }()
		synctest.Wait()
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		release()
		last, err := budget.Acquire(t.Context(), 32<<20)
		if err != nil {
			t.Fatal(err)
		}
		last()
		last()
		if budget.used != 0 || len(budget.waiting) != 0 {
			t.Fatal("projection reservation leaked")
		}
	})
}
