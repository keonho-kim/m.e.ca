// Purpose: Distinguish unacknowledged command presence from completed archival metadata.
// Pattern: Registry lifecycle and allocation regressions.
// Usage: Process tests use owned synthetic jobs without starting subprocesses.
// Related: internal/tools/local/process/registry.go, internal/tools/local/process/command_control.go.
package process

import (
	"fmt"
	"testing"
	"time"
)

func TestCommandPresenceIncludesUncollectedCompletion(t *testing.T) {
	done := make(chan struct{})
	r := &Registry{jobs: map[string]*job{"one": {state: State{ID: "one"}, done: done}}}
	if !r.HasCommands() {
		t.Fatal("running command was missing")
	}
	close(done)
	if !r.HasCommands() || r.ActiveID() != "" {
		t.Fatal("uncollected completion was confused with a running command")
	}
	if allocations := testing.AllocsPerRun(100, func() { r.HasCommands() }); allocations != 0 {
		t.Fatal("presence check allocated command snapshots", allocations)
	}
	r.Acknowledge("one")
	if r.HasCommands() {
		t.Fatal("acknowledged command remained pending")
	}
}

func TestCompletionMetadataExcludesRunningAndArchivingJobs(t *testing.T) {
	r := completionRegistry()
	var storage [MaxActive]Completion
	completed := r.AppendCompleted(storage[:0])
	if len(completed) != 2 || completed[0].ID != "0" || completed[1].ID != "2" || completed[1].Error != "archive failed" {
		t.Fatal("completion ordering or archival outcome was lost", completed)
	}
	if allocations := testing.AllocsPerRun(100, func() { r.AppendCompleted(storage[:0]) }); allocations != 0 {
		t.Fatal("bounded completion metadata allocated snapshots", allocations)
	}
	if !r.HasCommands() {
		t.Fatal("observing completions acknowledged them")
	}
}

func completionRegistry() *Registry {
	r := &Registry{jobs: make(map[string]*job)}
	for i := range MaxActive {
		id := fmt.Sprint(i)
		j := &job{state: State{ID: id, CallID: "call_" + id, Command: "check", Status: "exited"}, started: time.Now(), done: make(chan struct{})}
		if i == 2 {
			j.state.Error = "archive failed"
		}
		if i%2 == 0 {
			close(j.done)
		}
		r.jobs[id] = j
	}
	return r
}

func BenchmarkCommandMetadata(b *testing.B) {
	r := completionRegistry()
	b.Run("full_snapshot", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			r.List()
		}
	})
	b.Run("presence", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			r.HasCommands()
		}
	})
	b.Run("completed", func(b *testing.B) {
		var storage [MaxActive]Completion
		b.ReportAllocs()
		for b.Loop() {
			r.AppendCompleted(storage[:0])
		}
	})
}
