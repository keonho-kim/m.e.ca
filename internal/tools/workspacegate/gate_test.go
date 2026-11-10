// Purpose: Keep independent task mutations exclusive while allowing cooperative peers to share ownership.
// Pattern: Deterministic resource lifecycle tests.
// Usage: go test -race ./internal/tools/workspacegate.
// Related: internal/tools/workspacegate/gate.go, internal/tools/local/prepare.go.
package workspacegate

import (
	"context"
	"testing"
	"testing/synctest"
)

func TestTaskCohortKeepsOwnershipUntilJoinedRelease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var gate Gate
		first, releaseFirst := Bind(t.Context(), &gate)
		defer releaseFirst()
		second, releaseSecond := Bind(t.Context(), &gate)
		defer releaseSecond()
		if err := hold(t, first); err != nil {
			t.Fatal(err)
		}
		child, stop := context.WithCancel(first)
		defer stop()
		if err := hold(t, child); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- hold(t, second) }()
		synctest.Wait()
		select {
		case <-done:
			t.Fatal("another task entered the shared workspace")
		default:
		}
		releaseFirst()
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}

func TestWaitingTaskCancellationDoesNotReleaseAnotherOwner(t *testing.T) {
	var gate Gate
	owner, release := Bind(t.Context(), &gate)
	defer release()
	if err := hold(t, owner); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	waiting, closeWaiting := Bind(ctx, &gate)
	cancel()
	if err := hold(t, waiting); err != context.Canceled {
		t.Fatal(err)
	}
	closeWaiting()
	if err := hold(t, owner); err != nil {
		t.Fatal(err)
	}
}

func hold(t *testing.T, ctx context.Context) error {
	t.Helper()
	release, err := Acquire(ctx)
	if err == nil {
		t.Cleanup(release)
	}
	return err
}

func TestIdleTaskReleasesWorkspaceBeforeTaskEnds(t *testing.T) {
	var gate Gate
	first, closeFirst := Bind(t.Context(), &gate)
	defer closeFirst()
	second, closeSecond := Bind(t.Context(), &gate)
	defer closeSecond()
	prepared, err := Acquire(first)
	if err != nil {
		t.Fatal(err)
	}
	process, err := Acquire(first)
	if err != nil {
		t.Fatal(err)
	}
	prepared()
	ctx, cancel := context.WithCancel(second)
	cancel()
	if _, err := Acquire(ctx); err != context.Canceled {
		t.Fatal(err)
	}
	process()
	other, err := Acquire(second)
	if err != nil {
		t.Fatal(err)
	}
	other()
}
