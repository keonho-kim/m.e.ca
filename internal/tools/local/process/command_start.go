// Purpose: Admit and start one owned command before publishing its registry identity.
// Pattern: Joined command lifecycle.
// Usage: Registry.Start reserves capacity and installs one completion worker.
// Related: internal/tools/local/process/registry.go, internal/tools/local/process/command_finish.go.
package process

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"
)

func (r *Registry) Start(ctx context.Context, directory, command string, environment []string, callID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	active := 0
	for _, j := range r.jobs {
		select {
		case <-j.done:
		default:
			active++
		}
	}
	if active >= MaxActive {
		return "", fmt.Errorf("command capacity reached (%d active); use list_command_runs and wait_command or cancel_command", MaxActive)
	}
	owner := r.owner
	if owner == nil {
		owner = ctx
	}
	if err := owner.Err(); err != nil {
		return "", err
	}
	child, cancelCause := context.WithCancelCause(owner)
	// Owner cancellation closes Done before all children necessarily observe
	// its cause. Cleanup must not replace an already expired deadline.
	cancel := func() { cancelCause(context.Cause(owner)) }
	id := "cmd_" + rand.Text()
	j := &job{state: State{ID: id, CallID: callID, Command: command, Directory: directory, Status: "running"}, started: time.Now(), done: make(chan struct{}), cancel: cancel}
	j.stdout.directory, j.stderr.directory = r.CaptureDirectory, r.CaptureDirectory
	pending, err := start(child, directory, command, environment, &j.stdout, &j.stderr)
	if err != nil {
		cancel()
		return "", err
	}
	if r.jobs == nil {
		r.jobs = make(map[string]*job)
	}
	r.jobs[id] = j
	save := r.Save
	go r.finish(id, j, pending, save)
	return id, nil
}

// Updates is a latest-state notification, not a queue of output bytes.
