//go:build linux || darwin

// Purpose: Verify the full team command capacity consists of actual concurrent processes and joins on shutdown.
// Pattern: Eight isolated registries sharing one request context and output budget.
// Usage: go test -race ./internal/tools/local/process -run Test128Commands.
// Related: internal/tools/local/process/registry.go, internal/tools/local/process/projection.go.
package process

import (
	"os"
	"sync"
	"testing"
	"time"
)

func Test128CommandsRunAndJoinWithoutGlobalQueue(t *testing.T) {
	ctx := WithProjectionBudget(t.Context())
	var registries [8]Registry
	root := t.TempDir()
	defer func() {
		var joined sync.WaitGroup
		for i := range registries {
			joined.Go(registries[i].End)
		}
		joined.Wait()
	}()
	for i := range registries {
		r := &registries[i]
		r.Begin(ctx, time.Now().Add(time.Minute))
		for range 16 {
			if _, err := r.Start(ctx, root, "sleep 60", os.Environ(), "call"); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i := range registries {
		r := &registries[i]
		states := r.List()
		if len(states) != 16 || r.Available() != 0 {
			t.Fatal("lost bounded process capacity", len(states))
		}
		for _, state := range states {
			if state.Status != "running" {
				t.Fatal("command entry did not start an OS process", state)
			}
		}
	}
	var joined sync.WaitGroup
	for i := range registries {
		joined.Go(registries[i].End)
	}
	joined.Wait()
	for i := range registries {
		for _, state := range registries[i].List() {
			if state.Status != "interrupted" {
				t.Fatal("shutdown did not settle an owned process", state)
			}
			registries[i].Acknowledge(state.ID)
		}
		if registries[i].HasCommands() || registries[i].ActiveID() != "" {
			t.Fatal("shutdown retained active command metadata")
		}
	}
}
