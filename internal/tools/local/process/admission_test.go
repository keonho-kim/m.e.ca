// Purpose: Verify queued commands remain observable, cancellable and bounded before OS process start.
// Pattern: Controlled admission barrier with joined command lifetime.
// Usage: go test -race ./internal/tools/local/process -run TestWaitingCommand.
// Related: internal/tools/local/process/command_start.go, internal/tools/local/process/registry.go.
package process

import (
	"context"
	"os"
	"testing"
	"testing/synctest"
	"time"
)

func TestWaitingCommandReturnsWithinObservationWindowAndCancels(t *testing.T) {
	var r Registry
	r.Begin(t.Context(), time.Now().Add(time.Minute))
	defer r.End()
	entered := make(chan struct{})
	id, err := r.StartAdmitted(t.Context(), t.TempDir(), "printf never", os.Environ(), "queued", func(ctx context.Context) (func(), error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	state, err := r.Wait(t.Context(), id, 0)
	if err != nil || state.Status != "waiting" || len(r.List()) != 1 || r.Available() != MaxActive-1 {
		t.Fatalf("waiting command disappeared: %+v %v", state, err)
	}
	state, err = r.Cancel(t.Context(), id)
	if err != nil || state.Status == "waiting" || state.Status == "running" || r.Available() != MaxActive {
		t.Fatalf("cancel did not join waiting admission: %+v %v", state, err)
	}
}

func TestWaitingCommandInitialWindowIncludesAdmission(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var r Registry
		r.Begin(t.Context(), time.Now().Add(time.Hour))
		defer r.End()
		id, err := r.StartAdmitted(t.Context(), "/project", "printf never", nil, "queued", func(ctx context.Context) (func(), error) { <-ctx.Done(); return nil, ctx.Err() })
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		state, err := r.Wait(t.Context(), id, InitialWait)
		if err != nil || state.Status != "waiting" || time.Since(start) != InitialWait {
			t.Fatalf("admission escaped its observation window: %+v %v", state, err)
		}
		if _, err := r.Cancel(t.Context(), id); err != nil {
			t.Fatal(err)
		}
	})
}

func TestWaitingCommandEndJoinsAdmissionCleanup(t *testing.T) {
	var r Registry
	r.Begin(t.Context(), time.Now().Add(time.Minute))
	released := make(chan struct{})
	_, err := r.StartAdmitted(t.Context(), t.TempDir(), "printf never", os.Environ(), "queued", func(ctx context.Context) (func(), error) {
		<-ctx.Done()
		return func() { close(released) }, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	r.End()
	select {
	case <-released:
	default:
		t.Fatal("shutdown returned before queued cleanup")
	}
}
