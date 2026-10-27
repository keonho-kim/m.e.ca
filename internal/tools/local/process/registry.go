// Purpose: Own a bounded set of command lifetimes independently of individual wait calls.
// Pattern: Session registry with one waiter per command.
// Usage: The local executor binds a request, starts commands, snapshots and cancels its own jobs.
// Related: internal/tools/local/process/state.go, internal/tools/local/process/run_unix.go.
package process

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	MaxActive   = 4
	InitialWait = 30 * time.Second
	DefaultWait = 180 * time.Second
)

type Registry struct {
	mu    sync.Mutex
	owner context.Context
	stop  context.CancelFunc
	jobs  map[string]*job
	Save  func(string, Output, error) error
}

type job struct {
	mu             sync.Mutex
	state          State
	started        time.Time
	finished       time.Time
	done           chan struct{}
	cancel         context.CancelFunc
	stdout, stderr cappedOutput
}

func (r *Registry) Begin(ctx context.Context, deadline time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.owner, r.stop = context.WithDeadline(ctx, deadline)
}

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
	child, cancel := context.WithCancel(owner)
	id := "cmd_" + rand.Text()
	j := &job{state: State{ID: id, CallID: callID, Command: command, Directory: directory, Status: "running"}, started: time.Now(), done: make(chan struct{}), cancel: cancel}
	wait, err := start(child, directory, command, environment, &j.stdout, &j.stderr)
	if err != nil {
		cancel()
		return "", err
	}
	if r.jobs == nil {
		r.jobs = make(map[string]*job)
	}
	r.jobs[id] = j
	save := r.Save
	go func() {
		defer close(j.done)
		defer cancel()
		output, runErr := wait()
		var saveErr error
		if save != nil {
			saveErr = save(id, output, runErr)
		}
		j.stdout.release()
		j.stderr.release()
		j.mu.Lock()
		j.finished = time.Now()
		j.state.Status, j.state.ExitCode = output.Status, output.ExitCode
		if saveErr != nil {
			j.state.Error = saveErr.Error()
		}
		j.mu.Unlock()
	}()
	return id, nil
}

func (r *Registry) lookup(id string) (*job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	j := r.jobs[id]
	if j == nil {
		return nil, fmt.Errorf("unknown owned command_id %q; use list_command_runs", id)
	}
	return j, nil
}

func (r *Registry) Wait(ctx context.Context, id string, duration time.Duration) (State, error) {
	if err := ctx.Err(); err != nil {
		return State{}, err
	}
	j, err := r.lookup(id)
	if err != nil {
		return State{}, err
	}
	select {
	case <-j.done:
		return j.snapshot(true), nil
	default:
	}
	if duration <= 0 {
		return j.snapshot(true), nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-j.done:
		return j.snapshot(true), nil
	case <-timer.C:
		return j.snapshot(true), nil
	case <-ctx.Done():
		return j.snapshot(true), ctx.Err()
	}
}

func (r *Registry) Cancel(ctx context.Context, id string) (State, error) {
	j, err := r.lookup(id)
	if err != nil {
		return State{}, err
	}
	j.cancel()
	return r.Wait(ctx, id, 24*time.Hour)
}

func (r *Registry) List() []State {
	r.mu.Lock()
	defer r.mu.Unlock()
	states := make([]State, 0, len(r.jobs))
	for _, j := range r.jobs {
		states = append(states, j.snapshot(false))
	}
	sort.Slice(states, func(i, j int) bool { return states[i].ID < states[j].ID })
	return states
}

func (r *Registry) Acknowledge(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if j := r.jobs[id]; j != nil {
		select {
		case <-j.done:
			delete(r.jobs, id)
		default:
		}
	}
}

func (r *Registry) End() {
	r.mu.Lock()
	if r.stop != nil {
		r.stop()
		r.stop = nil
	}
	jobs := make([]*job, 0, len(r.jobs))
	for _, j := range r.jobs {
		j.cancel()
		jobs = append(jobs, j)
	}
	r.owner = nil
	r.mu.Unlock()
	for _, j := range jobs {
		<-j.done
	}
}
