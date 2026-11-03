// Purpose: Own one cooperative request, its membership and joined worker lifetimes.
// Pattern: Serialized state transitions with independent session workers.
// Usage: Request composition calls Run with a persistent room and per-session executor.
// Related: internal/agent/team/membership.go, internal/agent/team/worker.go.
package team

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"sync"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/work"
)

type participant struct {
	member          identity.Member
	wake            chan struct{}
	leaving, closed bool
}
type Room struct {
	mu           sync.Mutex
	memberMu     sync.Mutex
	logMu        sync.Mutex
	eventMu      sync.Mutex
	inputMu      sync.Mutex
	options      Options
	ctx          context.Context
	cancel       context.CancelCauseFunc
	workers      sync.WaitGroup
	changed      chan struct{}
	active       map[string]*participant
	result       Result
	input        string
	firstMessage int64
	latestPublic int64
	public       publicHistory
	latestUser   int64
	deadline     time.Time
	modelCalls   int
	finished     bool
	failures     error
	statistics   runrecord.UsageTotals
	runID        string
}

func Run(ctx context.Context, options Options, text string) (Result, error) {
	if options.Store == nil || options.Execute == nil || options.Emit == nil || options.MaxModelCalls < 1 || options.MaxSeconds < 1 {
		return Result{}, fmt.Errorf("team dependencies and request limits are required")
	}
	if options.Redact != nil {
		text = options.Redact(text)
	}
	started := time.Now()
	deadline := started.Add(time.Duration(options.MaxSeconds) * time.Second)
	ctx, stop := context.WithDeadline(ctx, deadline)
	defer stop()
	ctx, cancel := context.WithCancelCause(ctx)
	r := &Room{options: options, ctx: ctx, cancel: cancel, changed: make(chan struct{}, 1), active: map[string]*participant{}, input: text, runID: "run_" + rand.Text()}
	r.result.Started, r.deadline = started, deadline
	defer func() { cancel(nil); r.workers.Wait() }()
	if err := r.initialize(); err != nil {
		return r.result, err
	}
	// Only the user-input channel is checked here; public messages wake peers directly.
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			cancel(nil)
			r.workers.Wait()
			return r.complete(context.Cause(ctx))
		case <-tick.C:
			if err := r.admitUser(); err != nil {
				cancel(err)
				r.workers.Wait()
				return r.complete(err)
			}
		case <-r.changed:
		}
		r.mu.Lock()
		done := r.countLocked() == 0
		r.mu.Unlock()
		if done {
			r.workers.Wait()
			cause := context.Cause(ctx)
			cancel(nil)
			return r.complete(cause)
		}
	}
}

func (r *Room) initialize() error {
	r.memberMu.Lock()
	defer r.memberMu.Unlock()
	members, err := r.options.Store.TeamMembers(r.ctx, "", 2)
	if err != nil {
		return err
	}
	for len(members) < 2 {
		member, err := r.options.Store.CreateTeamAgent(r.ctx)
		if err != nil {
			return err
		}
		members = append(members, member)
	}
	message, err := r.publish(channel.Message{From: "user", Text: r.input})
	if err != nil {
		return err
	}
	r.firstMessage = message.Sequence
	initial := make([]*participant, 0, len(members))
	for _, member := range members {
		initial = append(initial, r.reserve(member))
	}
	for _, live := range initial {
		if err := r.launch(live); err != nil {
			return err
		}
	}
	return nil
}
func (r *Room) countLocked() int {
	n := 0
	for _, p := range r.active {
		if !p.closed {
			n++
		}
	}
	return n
}
func (r *Room) remainingLocked() int {
	n := 0
	for _, p := range r.active {
		if !p.closed && !p.leaving {
			n++
		}
	}
	return n
}
func (r *Room) notify() {
	select {
	case r.changed <- struct{}{}:
	default:
	}
}

func (r *Room) complete(cause error) (Result, error) {
	// Run joins all workers before this final snapshot and delivery.
	if cause == nil && r.result.Text == "" {
		cause = errors.Join(fmt.Errorf("all team sessions stopped before a final report"), r.failures)
	}
	status, end := runrecord.Answered, work.Completed
	if cause != nil {
		status, end = runrecord.Stopped, work.Failed
	}
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		status, end = runrecord.Interrupted, work.Interrupted
	}
	r.result.Status = status
	cp := runrecord.Checkpoint{ID: r.runID, Objective: r.input, Status: status, Phase: "finished", ModelCalls: r.modelCalls, End: &work.End{Status: end}}
	if cause != nil {
		cp.End.Reason = cause.Error()
	}
	statistics := r.statistics.Snapshot(time.Since(r.result.Started).Milliseconds(), true)
	event := runrecord.Event{Kind: "run_checkpoint", RunID: cp.ID, Checkpoint: &cp, Statistics: &statistics}
	if r.options.Record != nil {
		flush, cancel := context.WithTimeout(context.WithoutCancel(r.ctx), 5*time.Second)
		defer cancel()
		if err := r.options.Record(flush, event); err != nil {
			cause = errors.Join(cause, err)
			cp.Status, r.result.Status = runrecord.Stopped, runrecord.Stopped
			cp.End = &work.End{Status: work.Failed, Reason: cause.Error()}
		}
	}
	cause = errors.Join(cause, r.options.Emit(event))
	if cause != nil && r.result.Status == runrecord.Answered {
		r.result.Status = runrecord.Stopped
	}
	return r.result, cause
}
