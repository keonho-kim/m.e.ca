// Purpose: Share bounded token aggregates across handles to the same append-only session ledger.
// Pattern: Identity-checked repository projection cache.
// Usage: Sessions lends aggregates to Store; eviction retains no database handles or event bodies.
// Related: internal/storage/sqlite/catalog.go, internal/storage/sqlite/usage.go.
package sqlite

import (
	"os"
	"sync"

	"github.com/keonho-kim/meca/internal/agent/model"
)

type usageProjection struct {
	mu       sync.Mutex
	identity os.FileInfo
	sequence int64
	total    model.TokenUsage
}

type usageProjections struct {
	mu      sync.Mutex
	entries map[string]*usageProjection
	order   [128]string
	next    int
}

func (c *usageProjections) forFile(path string, identity os.FileInfo) *usageProjection {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry := c.entries[path]; entry != nil && os.SameFile(entry.identity, identity) {
		return entry
	}
	if c.entries == nil {
		c.entries = make(map[string]*usageProjection)
	}
	if c.entries[path] == nil {
		delete(c.entries, c.order[c.next])
		c.order[c.next] = path
		c.next = (c.next + 1) % len(c.order)
	}
	entry := &usageProjection{identity: identity}
	c.entries[path] = entry
	return entry
}

func (c *usageProjections) forget(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, path)
	for i, entry := range c.order {
		if entry == path {
			c.order[i] = ""
		}
	}
}
