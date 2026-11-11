// Purpose: Verify canceled usage readers do not wait for another aggregate query.
// Pattern: Deterministic projection admission regression tests.
// Usage: go test -race ./internal/storage/sqlite -run TestUsageAdmission.
// Related: internal/storage/sqlite/usage_cache.go, internal/storage/sqlite/usage.go.
package sqlite

import (
	"context"
	"errors"
	"testing"
)

func TestUsageAdmissionCancellation(t *testing.T) {
	var projection usageProjection
	release, err := projection.acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := projection.acquire(ctx); done <- err }()
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal("reader ignored cancellation while another query owns the projection", err)
	}
	release()
	release, err = projection.acquire(t.Context())
	if err != nil {
		t.Fatal("canceled admission retained its permit", err)
	}
	release()
}
