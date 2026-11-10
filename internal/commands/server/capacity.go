// Purpose: Share the running-task allowance across project runtimes in one server.
// Pattern: Explicit resource lease.
// Usage: Start reserves before durable admission and execution releases after completion.
// Related: internal/commands/server/tasks.go, internal/commands/server/execution.go.
package server

import "sync"

type Capacity struct {
	mu            sync.Mutex
	limit, active int
}

func NewCapacity(limit int) *Capacity { return &Capacity{limit: max(1, limit)} }

func (c *Capacity) reserve() (func(), bool) {
	if c == nil {
		return func() {}, true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active >= c.limit {
		return nil, false
	}
	c.active++
	return sync.OnceFunc(func() { c.mu.Lock(); c.active--; c.mu.Unlock() }), true
}

func (c *Capacity) Active() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.active
}
