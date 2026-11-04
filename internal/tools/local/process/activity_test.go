// Purpose: Keep execution admission independent of output snapshots and list allocations.
// Pattern: Registry lifecycle and allocation regression tests.
// Usage: Process tests use owned synthetic jobs without launching subprocesses.
// Related: internal/tools/local/process/command_control.go, internal/tools/local/commands.go.
package process

import (
	"fmt"
	"testing"
	"time"
)

func TestActiveCommandRemainsActiveThroughArchivalWithoutAllocating(t *testing.T) {
	done := make(chan struct{})
	r := &Registry{jobs: map[string]*job{"command": {state: State{ID: "command", Status: "exited"}, done: done}}}
	if id := r.ActiveID(); id != "command" {
		t.Fatal("terminal status released admission before archival completed", id)
	}
	if allocations := testing.AllocsPerRun(100, func() { r.ActiveID() }); allocations != 0 {
		t.Fatal("active-command lookup allocated a snapshot", allocations)
	}
	close(done)
	if id := r.ActiveID(); id != "" {
		t.Fatal("settled command remained active", id)
	}
}

func BenchmarkCommandAdmission(b *testing.B) {
	r := &Registry{jobs: make(map[string]*job)}
	for i := range 4 {
		id := fmt.Sprint(i)
		r.jobs[id] = &job{state: State{ID: id, Status: "running"}, started: time.Now(), done: make(chan struct{})}
	}
	b.Run("snapshot", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			r.List()
		}
	})
	b.Run("active_id", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			r.ActiveID()
		}
	})
}
