// Purpose: Verify wait expiry, owned identities, cancellation and cleanup independently of command duration.
// Pattern: Process lifecycle integration tests.
// Usage: go test -race ./internal/tools/local/process -run TestRegistry.
// Related: internal/tools/local/process/registry.go, internal/tools/local/process/run_unix.go.
package process

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"
)

func TestRegistryInitialAndFollowupWaitWindows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := time.Now()
		cancelled := false
		j := &job{state: State{ID: "cmd_owned", Status: "running"}, started: started, done: make(chan struct{}), cancel: func() { cancelled = true }}
		defer close(j.done)
		registry := Registry{jobs: map[string]*job{"cmd_owned": j}}
		state, err := registry.Wait(context.Background(), "cmd_owned", InitialWait)
		if err != nil || state.Status != "running" || time.Since(started) != 30*time.Second || cancelled {
			t.Fatalf("initial wait terminated or waited beyond 30 seconds: %+v %v", state, err)
		}
		followup := time.Now()
		state, err = registry.Wait(context.Background(), "cmd_owned", DefaultWait)
		if err != nil || state.Status != "running" || time.Since(followup) != 180*time.Second || cancelled {
			t.Fatalf("followup wait changed its independent default: %+v %v", state, err)
		}
	})
}

func TestRegistryWaitDoesNotCancel(t *testing.T) {
	root := t.TempDir()
	release := filepath.Join(root, "release")
	var registry Registry
	registry.Begin(context.Background(), time.Now().Add(time.Minute))
	t.Cleanup(func() { registry.End() })
	registry.Save = func(string, Output, error) error { return nil }
	id, err := registry.Start(context.Background(), root, "printf started; while [ ! -f release ]; do sleep 0.02; done; printf finished", os.Environ(), "call_1")
	if err != nil {
		t.Fatal(err)
	}
	state, err := registry.Wait(context.Background(), id, 0)
	if err != nil || state.Status != "running" {
		t.Fatalf("initial state %+v: %v", state, err)
	}
	if err := os.WriteFile(release, nil, 0600); err != nil {
		t.Fatal(err)
	}
	state, err = registry.Wait(context.Background(), id, time.Second*5)
	if err != nil || state.Status != "exited" || state.ExitCode != 0 {
		t.Fatalf("final state %+v: %v", state, err)
	}
}

func TestRegistryOwnsCancellationAndCapacity(t *testing.T) {
	var registry Registry
	registry.Begin(context.Background(), time.Now().Add(time.Minute))
	registry.Save = func(string, Output, error) error { return nil }
	t.Cleanup(func() { registry.End() })
	for range MaxActive {
		if _, err := registry.Start(context.Background(), t.TempDir(), "sleep 60", os.Environ(), "call"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := registry.Start(context.Background(), t.TempDir(), "true", os.Environ(), "extra"); err == nil {
		t.Fatal("unbounded commands accepted")
	}
	if _, err := registry.Wait(context.Background(), "1", 0); err == nil {
		t.Fatal("OS PID accepted")
	}
	for _, state := range registry.List() {
		result, err := registry.Cancel(context.Background(), state.ID)
		if err != nil || result.Status != "interrupted" {
			t.Fatalf("cancel %+v: %v", result, err)
		}
	}
	registry.End()
	for _, state := range registry.List() {
		if state.Status == "running" {
			t.Fatal("command survived End")
		}
	}
}

func TestRegistryWatcherCancellationLeavesCommandRunning(t *testing.T) {
	var registry Registry
	registry.Begin(context.Background(), time.Now().Add(time.Minute))
	registry.Save = func(string, Output, error) error { return nil }
	t.Cleanup(func() { registry.End() })
	id, err := registry.Start(context.Background(), t.TempDir(), "sleep 60", os.Environ(), "call")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := registry.Wait(ctx, id, time.Minute); err == nil {
		t.Fatal("cancelled watcher waited")
	}
	state, err := registry.Wait(context.Background(), id, 0)
	if err != nil || state.Status != "running" {
		t.Fatalf("watcher killed process %+v: %v", state, err)
	}
}
