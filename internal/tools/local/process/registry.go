// Purpose: Own a bounded set of command lifetimes independently of individual wait calls.
// Pattern: Session registry with one waiter per command.
// Usage: The local executor binds a request, starts commands, snapshots and cancels its own jobs.
// Related: internal/tools/local/process/state.go, internal/tools/local/process/run_unix.go.
package process

import (
	"context"
	"sync"
	"time"
)

const (
	// MaxActive bounds waiting, running and finalizing commands owned by one agent.
	MaxActive   = 16
	InitialWait = 30 * time.Second
	DefaultWait = 180 * time.Second
)

type Registry struct {
	CaptureDirectory string
	mu               sync.Mutex
	projection       *ProjectionBudget
	owner            context.Context
	stop             context.CancelFunc
	jobs             map[string]*job
	updates          chan struct{}
	Save             func(id, command string, output Output, runErr error) error
}

type job struct {
	mu             sync.Mutex
	state          State
	started        time.Time
	finished       time.Time
	done           chan struct{}
	launched       chan struct{}
	cancel         context.CancelFunc
	stdout, stderr cappedOutput
}

// One worker waits, then projects output under the request-shared byte budget.
type startedCommand struct {
	wait   func() error
	output func(error) (Output, error)
}

func (r *Registry) Begin(ctx context.Context, deadline time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.owner, r.stop = context.WithDeadline(ctx, deadline)
	if shared, ok := ctx.Value(projectionKey{}).(*ProjectionBudget); ok {
		r.projection = shared
	}
	if r.updates == nil {
		r.updates = make(chan struct{}, 1)
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
