// Purpose: Register bounded command ownership before admission and OS launch outside the registry lock.
// Pattern: Joined command lifecycle.
// Usage: Registry.Start reserves capacity and installs one completion worker.
// Related: internal/tools/local/process/registry.go, internal/tools/local/process/command_finish.go.
package process

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

// Start reserves a job before launching its process outside the registry lock.
func (r *Registry) Start(ctx context.Context, directory, command string, environment []string, callID string) (string, error) {
	id, j, err := r.register(ctx, directory, command, environment, callID, nil, true)
	if err != nil {
		return id, err
	}
	<-j.launched
	return id, j.state.Failure
}

// StartAdmitted includes resource admission in the observable command lifetime.
func (r *Registry) StartAdmitted(ctx context.Context, directory, command string, environment []string, callID string, admit func(context.Context) (func(), error)) (string, error) {
	id, _, err := r.register(ctx, directory, command, environment, callID, admit, false)
	return id, err
}

func (r *Registry) register(ctx context.Context, directory, command string, environment []string, callID string, admit func(context.Context) (func(), error), waitStart bool) (string, *job, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	r.mu.Lock()
	if r.availableLocked() == 0 {
		r.mu.Unlock()
		return "", nil, fmt.Errorf("command capacity reached (%d active); use list_command_runs and wait_command or cancel_command", MaxActive)
	}
	owner := r.owner
	if owner == nil {
		owner = ctx
	}
	if err := owner.Err(); err != nil {
		r.mu.Unlock()
		return "", nil, err
	}
	child, cancelCause := context.WithCancelCause(owner)
	cancel := func() { cancelCause(context.Cause(owner)) }
	id := "cmd_" + rand.Text()
	j := &job{state: State{ID: id, CallID: callID, Command: command, Directory: directory, Status: "waiting"}, started: time.Now(), done: make(chan struct{}), cancel: cancel}
	if waitStart {
		j.launched = make(chan struct{})
	}
	j.stdout.directory, j.stderr.directory = r.CaptureDirectory, r.CaptureDirectory
	if r.jobs == nil {
		r.jobs = make(map[string]*job)
	}
	r.jobs[id] = j
	save := r.Save
	r.mu.Unlock()
	go r.launch(child, id, j, environment, admit, save)
	return id, j, nil
}

func (r *Registry) launch(ctx context.Context, id string, j *job, environment []string, admit func(context.Context) (func(), error), save func(string, string, Output, error) error) {
	signaled := false
	launched := func() {
		if j.launched != nil && !signaled {
			close(j.launched)
			signaled = true
		}
	}
	defer launched()
	var release func()
	defer func() {
		if release != nil {
			release()
		}
		j.cancel()
		close(j.done)
		r.notify()
	}()
	var err error
	if admit != nil {
		release, err = admit(ctx)
	}
	if err == nil {
		err = ctx.Err()
	}
	var pending startedCommand
	if err == nil {
		pending, err = start(ctx, j.state.Directory, j.state.Command, environment, &j.stdout, &j.stderr)
	}
	if err != nil {
		output := Output{Status: "execution_failed", ExitCode: -1}
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			output.Status = "interrupted"
			if errors.Is(context.Cause(ctx), context.DeadlineExceeded) {
				output.Status = "timeout"
			}
		}
		var archiveErr error
		if save != nil {
			archiveErr = save(id, j.state.Command, output, err)
		}
		j.mu.Lock()
		j.finished = time.Now()
		j.state.Status = output.Status
		j.state.ExitCode = -1
		j.state.Failure = err
		if archiveErr != nil {
			j.state.Error = archiveErr.Error()
		}
		j.mu.Unlock()
		return
	}
	j.mu.Lock()
	j.state.Status = "running"
	j.mu.Unlock()
	launched()
	r.finish(id, j, pending, save)
}

func (r *Registry) availableLocked() int {
	active := 0
	for _, j := range r.jobs {
		select {
		case <-j.done:
		default:
			active++
		}
	}
	return max(0, MaxActive-active)
}

// Available is a planning hint; StartAdmitted performs authoritative admission.
func (r *Registry) Available() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.availableLocked()
}
