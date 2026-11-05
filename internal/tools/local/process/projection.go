// Purpose: Bound simultaneous large output materialization across all commands in one request.
// Pattern: Shared weighted admission with a small-output fast path.
// Usage: Team request context shares one budget; standalone registries own an equivalent budget.
// Related: internal/tools/local/process/command_finish.go, internal/tools/local/command_live.go.
package process

import (
	"context"
	"sync"
)

const projectionBytes = 64 << 20

type projectionKey struct{}
type ProjectionBudget struct {
	mu            sync.Mutex
	changed       *sync.Cond
	used          int64
	next, serving uint64
}

func WithProjectionBudget(ctx context.Context) context.Context {
	return context.WithValue(ctx, projectionKey{}, &ProjectionBudget{})
}

// Acquire accounts for output strings, redaction and JSON encoding together.
// Pipe capture and process execution never hold this permit.
func projectionCost(bytes int64) int64 {
	if bytes <= 64<<10 {
		return 0
	}
	return min(int64(projectionBytes), bytes*4)
}
func (b *ProjectionBudget) release(cost int64) func() {
	if cost == 0 {
		return func() {}
	}
	return func() {
		b.mu.Lock()
		b.used -= cost
		if b.changed != nil {
			b.changed.Broadcast()
		}
		b.mu.Unlock()
	}
}
func (b *ProjectionBudget) Acquire(bytes int64) func() {
	cost := projectionCost(bytes)
	if cost == 0 {
		return func() {}
	}
	b.mu.Lock()
	ticket := b.next
	b.next++
	for ticket != b.serving || b.used+cost > projectionBytes {
		if b.changed == nil {
			b.changed = sync.NewCond(&b.mu)
		}
		b.changed.Wait()
	}
	b.used += cost
	b.serving++
	if b.next != b.serving && b.changed != nil {
		b.changed.Broadcast()
	}
	b.mu.Unlock()
	return b.release(cost)
}

// TryAcquire keeps live status observations from waiting behind output archival.
func (b *ProjectionBudget) TryAcquire(bytes int64) (func(), bool) {
	cost := projectionCost(bytes)
	if cost == 0 {
		return func() {}, true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.next != b.serving || b.used+cost > projectionBytes {
		return nil, false
	}
	b.used += cost
	return b.release(cost), true
}

func (r *Registry) ProjectionBudget() *ProjectionBudget {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.projection == nil {
		r.projection = &ProjectionBudget{}
	}
	return r.projection
}

func (output *cappedOutput) retainedBytes() int64 {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.retainedBytesLocked()
}
func (output *cappedOutput) retainedBytesLocked() int64 {
	if output.spill != nil {
		return int64(output.spill.head + output.spill.tail + 32)
	}
	return int64(output.head.Len() + len(output.tail) + 32)
}
