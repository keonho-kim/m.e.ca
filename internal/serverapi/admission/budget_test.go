// Purpose: Verify bounded admission, canceled waiters and reusable idempotent releases.
// Pattern: Deterministic resource-lifetime tests.
// Usage: go test -race ./internal/serverapi/admission.
// Related: internal/serverapi/admission/budget.go.
package admission

import (
	"context"
	"errors"
	"testing"
)

func TestBudgetBoundsCancellationAndRelease(t *testing.T) {
	b := New(10)
	release, ok := b.Try(7)
	if !ok {
		t.Fatal("initial reservation failed")
	}
	if extra, ok := b.Try(4); ok {
		extra()
		t.Fatal("budget exceeded")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := b.Acquire(ctx, 4); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	release()
	release()
	all, err := b.Acquire(t.Context(), 10)
	if err != nil {
		t.Fatal("reservation was not released", err)
	}
	all()
	if _, err := b.Acquire(t.Context(), 11); !errors.Is(err, ErrCapacity) {
		t.Fatal(err)
	}
}
