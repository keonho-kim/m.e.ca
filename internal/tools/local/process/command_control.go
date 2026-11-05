// Purpose: Inspect, wait and cancel commands through their existing registry ownership.
// Pattern: Joined command lifecycle.
// Usage: Tool calls observe job snapshots and never launch another waiter.
// Related: internal/tools/local/process/registry.go, internal/tools/local/process/state.go.
package process

import (
	"context"
	"fmt"
	"sort"
	"time"
)

func (r *Registry) Updates() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updates == nil {
		r.updates = make(chan struct{}, 1)
	}
	return r.updates
}

func (r *Registry) notify() {
	r.mu.Lock()
	defer r.mu.Unlock()
	select {
	case r.updates <- struct{}{}:
	default:
	}
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

// Done closes after process exit, output archival and capture cleanup.
func (r *Registry) Done(id string) (<-chan struct{}, error) {
	j, err := r.lookup(id)
	if err != nil {
		return nil, err
	}
	return j.done, nil
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
		return j.snapshot(), nil
	default:
	}
	if duration <= 0 {
		return j.snapshot(), nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-j.done:
		return j.snapshot(), nil
	case <-timer.C:
		return j.snapshot(), nil
	case <-ctx.Done():
		return j.snapshot(), ctx.Err()
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

func (r *Registry) List() []State { return r.AppendStates(nil) }

// AppendStates snapshots references under the lock, then formats outside it.
func (r *Registry) AppendStates(states []State) []State {
	r.mu.Lock()
	var storage [MaxActive]*job
	jobs := storage[:0]
	for _, j := range r.jobs {
		jobs = append(jobs, j)
	}
	r.mu.Unlock()
	from := len(states)
	for _, j := range jobs {
		states = append(states, j.snapshot())
	}
	sort.Slice(states[from:], func(i, j int) bool { return states[from+i].ID < states[from+j].ID })
	return states
}

// ActiveID checks actual owned completion without allocating or reading output metadata.
func (r *Registry) ActiveID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, j := range r.jobs {
		select {
		case <-j.done:
		default:
			return id
		}
	}
	return ""
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
