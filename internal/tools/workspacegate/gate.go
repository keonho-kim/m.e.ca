// Purpose: Serialize prepared workspace effects and live subprocesses while allowing idle tasks to progress.
// Pattern: Cancellable cohort leases with FIFO admission and explicit effect-owned release.
// Usage: Server binds a cohort; local preparation and process launch acquire scoped holds.
// Related: internal/tools/local/prepare.go, internal/tools/local/process/command_start.go.
package workspacegate

import (
	"context"
	"slices"
	"sync"
)

type key struct{}
type Gate struct {
	mu      sync.Mutex
	owner   *lease
	changed chan struct{}
	waiting []*waiter
}
type lease struct {
	gate   *Gate
	closed bool
	holds  int
}
type waiter struct{ owner *lease }

func Bind(ctx context.Context, gate *Gate) (context.Context, func()) {
	l := &lease{gate: gate}
	return context.WithValue(ctx, key{}, l), sync.OnceFunc(func() {
		gate.mu.Lock()
		defer gate.mu.Unlock()
		l.closed = true
		if gate.owner == l {
			gate.owner = nil
		}
		gate.wake()
	})
}
func (g *Gate) wake() {
	if g.changed != nil {
		close(g.changed)
		g.changed = nil
	}
}

func Acquire(ctx context.Context) (func(), error) {
	l, ok := ctx.Value(key{}).(*lease)
	if !ok {
		return func() {}, nil
	}
	g := l.gate
	g.mu.Lock()
	defer g.mu.Unlock()
	ticket := &waiter{owner: l}
	queued := false
	remove := func() {
		if queued {
			at := slices.Index(g.waiting, ticket)
			if at >= 0 {
				g.waiting = slices.Delete(g.waiting, at, at+1)
			}
			queued = false
			g.wake()
		}
	}
	for {
		if err := ctx.Err(); err != nil {
			remove()
			return nil, err
		}
		if l.closed {
			remove()
			return nil, context.Canceled
		}
		if g.owner == l || g.owner == nil && (len(g.waiting) == 0 || g.waiting[0] == ticket) {
			remove()
			g.owner = l
			l.holds++
			return sync.OnceFunc(func() {
				g.mu.Lock()
				defer g.mu.Unlock()
				l.holds--
				if g.owner == l && l.holds == 0 {
					g.owner = nil
					g.wake()
				}
			}), nil
		}
		if !queued {
			g.waiting = append(g.waiting, ticket)
			queued = true
		}
		if g.changed == nil {
			g.changed = make(chan struct{})
		}
		changed := g.changed
		g.mu.Unlock()
		select {
		case <-ctx.Done():
		case <-changed:
		}
		g.mu.Lock()
	}
}
