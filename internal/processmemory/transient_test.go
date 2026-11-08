// Purpose: Verify aggregate reply ownership and cancellable transient admission without large allocations.
// Pattern: Resource lifecycle contract tests.
// Usage: go test -race ./internal/processmemory.
// Related: internal/processmemory/transient.go.
package processmemory

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
)

func TestReplyLeasesShareAndReleaseCapacity(t *testing.T) {
	first, second := NewReplyLease(), NewReplyLease()
	defer first.Close()
	defer second.Close()
	if err := first.Resize(ReplyBytes); err != nil {
		t.Fatal(err)
	}
	if err := second.Resize(1); err == nil {
		t.Fatal("concurrent reply exceeded shared capacity")
	}
	first.Close()
	first.Close()
	if err := second.Resize(ReplyBytes); err != nil {
		t.Fatal("closed reply retained its charge", err)
	}
	if err := first.Resize(1); err == nil {
		t.Fatal("closed lease was reused")
	}
}

func TestTransientWaitCancellationDoesNotConsumeCapacity(t *testing.T) {
	budget := &transientBudget{maximum: 10}
	release, err := budget.acquire(t.Context(), 10)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := budget.acquire(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	release()
	release()
	again, err := budget.acquire(t.Context(), 10)
	if err != nil {
		t.Fatal(err)
	}
	again()
	if budget.used != 0 {
		t.Fatal("transient reservation leaked", budget.used)
	}
}

func TestTransientBlockedWaiterCanBeCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		budget := &transientBudget{maximum: 10}
		release, err := budget.acquire(t.Context(), 10)
		if err != nil {
			t.Fatal(err)
		}
		defer release()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		finished := make(chan error, 1)
		go func() { _, err := budget.acquire(ctx, 1); finished <- err }()
		synctest.Wait()
		cancel()
		if err := <-finished; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if budget.used != 10 {
			t.Fatal("canceled waiter changed another reservation", budget.used)
		}
	})
}
