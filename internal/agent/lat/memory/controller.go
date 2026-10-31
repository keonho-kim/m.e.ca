// Purpose: Maintain one independently derived review memo and one coalesced summary job.
// Pattern: Runtime-owned state machine with transactional publication.
// Usage: Only the runtime owner prepares, polls and commits; the job owns model I/O.
// Related: internal/agent/lat/memory/packet.go, internal/agent/lat/memory/worker.go.
package memory

import (
	"context"
	"errors"
	"fmt"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type Controller struct {
	Repository Repository
	Session    string
	Redact     func(string) string
	state      State
	loaded     bool
	job        *job
	through    int64
	failed     error
	mark       Mark
	markAfter  int64
}

func (c *Controller) loadState(ctx context.Context) (State, error) {
	if c.Repository == nil || c.Redact == nil {
		return State{}, fmt.Errorf("independent review memory repository is unavailable")
	}
	if !c.loaded {
		state, err := c.Repository.LATMemory(ctx, c.Session)
		if err != nil {
			return State{}, err
		}
		if err = state.Check(); err != nil {
			return State{}, err
		}
		c.state, c.loaded = state, true
	}
	return c.state, nil
}

// Prepare reads only one bounded prefix; updates while active are coalesced by
// reading the newest source boundary when the single slot next becomes free.
func (c *Controller) Prepare(ctx context.Context, instructions string, window, output int, force bool) (Plan, error) {
	if c.job != nil {
		return Plan{}, nil
	}
	if c.failed != nil {
		return Plan{}, c.failed
	}
	state, err := c.loadState(ctx)
	if err != nil {
		return Plan{}, err
	}
	mark, err := c.watermark(ctx, state)
	if err != nil {
		return Plan{}, err
	}
	budget, _, err := inputBudget(instructions, window, output)
	if err != nil {
		return Plan{}, err
	}
	if !force && mark.Bytes < int64(min(TargetBytes, budget/2)) || mark.Bytes == 0 {
		return Plan{}, nil
	}
	return c.plan(ctx, state, mark, instructions, window, output)
}

func (c *Controller) Start(ctx context.Context, plan Plan, generator modelapi.Generator, reserve func() bool) error {
	if c.job != nil || plan.Through <= c.state.Through || generator == nil {
		return fmt.Errorf("invalid independent review memory job")
	}
	if !reserve() {
		return fmt.Errorf("LAT memory model-call budget exhausted")
	}
	c.through = plan.Through
	c.job = start(ctx, plan.Request, generator, reserve)
	return nil
}

func (c *Controller) Active() bool { return c != nil && c.job != nil }
func (c *Controller) Done() <-chan struct{} {
	if !c.Active() {
		return nil
	}
	return c.job.done
}

func (c *Controller) Cancel() {
	if c.Active() {
		c.job.stop()
	}
}

// Poll never replaces a successful memo on generation or publication failure.
// Receipts are returned even when the memo could not be installed.
func (c *Controller) Poll(ctx context.Context) (Update, error) {
	if !c.Active() {
		return Update{}, nil
	}
	update := c.job.drain()
	if !update.Done {
		return update, nil
	}
	<-c.job.done
	defer func() { c.job = nil }()
	if c.job.err != nil {
		if !errors.Is(c.job.err, context.Canceled) {
			c.failed = c.job.err
		}
		return update, c.job.err
	}
	next := State{Revision: c.state.Revision + 1, Through: c.through, Text: c.Redact(c.job.text)}
	if err := next.Check(); err != nil {
		c.failed = err
		return update, err
	}
	if err := c.Repository.InstallLATMemory(ctx, c.Session, c.state.Revision, next); err != nil {
		c.failed = err
		return update, err
	}
	c.state = next
	return update, nil
}

// Tail preserves every uncompressed user instruction in a bounded review packet.
// An oversized tail requests another compaction, rather than omitting records.
func (c *Controller) Tail(ctx context.Context, budget int, current string) ([]byte, error) {
	state, err := c.loadState(ctx)
	if err != nil {
		return nil, err
	}
	mark, err := c.watermark(ctx, state)
	if err != nil {
		return nil, err
	}
	return c.tail(ctx, state, mark, budget, current)
}

func (c *Controller) watermark(ctx context.Context, state State) (Mark, error) {
	if c.markAfter != state.Through || c.mark.Through < state.Through {
		c.mark, c.markAfter = Mark{Through: state.Through}, state.Through
	}
	delta, err := c.Repository.LATWatermark(ctx, c.Session, c.mark.Through)
	if err != nil {
		return Mark{}, err
	}
	if delta.Through < c.mark.Through || delta.Bytes < 0 || delta.LatestUser > delta.Through {
		return Mark{}, fmt.Errorf("invalid independent review source watermark")
	}
	c.mark.Through = delta.Through
	c.mark.Bytes += delta.Bytes
	c.mark.LatestUser = max(c.mark.LatestUser, delta.LatestUser)
	return c.mark, nil
}
