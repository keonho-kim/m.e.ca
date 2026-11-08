// Purpose: Avoid replaying unclassified failures and preserve errors concurrent with cancellation.
// Pattern: Deterministic retry contract tests.
// Usage: go test ./internal/provider/transport -run TestRetryCause.
// Related: internal/provider/transport/retry.go.
package transport

import (
	"context"
	"errors"
	"testing"
)

func TestRetryCauseUnknownIsNotReplayed(t *testing.T) {
	client := New("", "")
	defer client.Close()
	calls := 0
	cause := errors.New("program invariant failed")
	err := client.Attempts(t.Context(), func() error { calls++; return cause }, nil)
	if calls != 1 || !errors.Is(err, cause) {
		t.Fatal(calls, err)
	}
}

func TestRetryCauseSurvivesCancellation(t *testing.T) {
	client := New("", "")
	defer client.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cause := errors.New("consumer commit failed")
	err := client.Attempts(ctx, func() error { cancel(); return &ConsumerError{Cause: cause} }, nil)
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
