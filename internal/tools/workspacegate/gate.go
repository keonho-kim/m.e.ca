// Purpose: Serialize overlapping file effects and unrestricted subprocesses with cancellable ownership.
// Pattern: Cohort leases with FIFO admission among conflicting scopes.
// Usage: Server binds canonical file scopes; agent preparation and processes use unrestricted cohorts.
// Related: internal/tools/local/prepare.go, internal/tools/local/process/command_start.go.
package workspacegate

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
)

type key struct{}
type Gate struct {
	mu      sync.Mutex
	owners  map[*lease]struct{}
	changed chan struct{}
	waiting []*waiter
}
type lease struct {
	gate   *Gate
	closed bool
	holds  int
	paths  []string
}
type waiter struct{ owner *lease }

func Bind(ctx context.Context, gate *Gate) (context.Context, func()) {
	return BindPaths(ctx, gate, nil)
}

// BindPaths requires canonical absolute paths checked again at the effect boundary.
// Empty or invalid scopes reserve the whole filesystem, as unrestricted commands do.
func BindPaths(ctx context.Context, gate *Gate, paths []string) (context.Context, func()) {
	for _, path := range paths {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			paths = nil
			break
		}
	}
	l := &lease{gate: gate, paths: slices.Clone(paths)}
	return context.WithValue(ctx, key{}, l), sync.OnceFunc(func() {
		gate.mu.Lock()
		defer gate.mu.Unlock()
		l.closed = true
		delete(gate.owners, l)
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
		if g.available(l, ticket) {
			remove()
			if g.owners == nil {
				g.owners = make(map[*lease]struct{})
			}
			g.owners[l] = struct{}{}
			l.holds++
			return sync.OnceFunc(func() {
				g.mu.Lock()
				defer g.mu.Unlock()
				l.holds--
				if l.holds == 0 {
					delete(g.owners, l)
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

func (g *Gate) available(owner *lease, ticket *waiter) bool {
	if _, held := g.owners[owner]; held {
		return true
	}
	for active := range g.owners {
		if overlaps(owner.paths, active.paths) {
			return false
		}
	}
	for _, waiting := range g.waiting {
		if waiting == ticket {
			break
		}
		if overlaps(owner.paths, waiting.owner.paths) {
			return false
		}
	}
	return true
}

func overlaps(first, second []string) bool {
	if len(first) == 0 || len(second) == 0 {
		return true
	}
	for _, a := range first {
		for _, b := range second {
			relative, err := filepath.Rel(a, b)
			if err == nil && filepath.IsLocal(relative) {
				return true
			}
			relative, err = filepath.Rel(b, a)
			if err == nil && filepath.IsLocal(relative) {
				return true
			}
		}
	}
	return false
}
