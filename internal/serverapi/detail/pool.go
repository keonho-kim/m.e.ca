// Purpose: Retain independently paged originals for concurrent views under one total cache budget.
// Pattern: Four-entry LRU with pinned readers and shared compressed storage admission.
// Usage: API owns one Cache; session cleanup invalidates matching documents without touching other views.
// Related: internal/serverapi/detail/cache.go, internal/serverapi/detail/chunks.go.
package detail

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"sync"
)

type cacheKey struct {
	file, revision string
	session        string
	source         server.DetailSource
}
type cacheSlot struct {
	key     cacheKey
	value   *documentCache
	used    uint64
	readers int
	closing int
}
type Cache struct {
	epoch  string
	mu     sync.Mutex
	slots  []*cacheSlot
	clock  uint64
	budget spoolBudget
}

func NewCache() *Cache { return &Cache{epoch: rand.Text()} }

func (c *Cache) borrow(request Request) (*cacheSlot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clock++
	key := cacheKey{session: request.Session, source: request.Source, file: request.File, revision: request.FileRevision}
	var candidate *cacheSlot
	for _, slot := range c.slots {
		if slot.key == key {
			if slot.closing > 0 {
				return nil, faults.New(diagcatalog.ResourceExhausted, diagcatalog.ServerOriginalReadBusy)
			}
			slot.readers++
			slot.used = c.clock
			return slot, nil
		}
		if slot.readers == 0 && (candidate == nil || slot.used < candidate.used) {
			candidate = slot
		}
	}
	if len(c.slots) < 4 {
		candidate = &cacheSlot{value: newDocumentCache(&c.budget)}
		c.slots = append(c.slots, candidate)
	} else if candidate == nil {
		return nil, faults.New(diagcatalog.ResourceExhausted, diagcatalog.DetailSpoolBudget)
	} else if err := candidate.value.Close(); err != nil {
		return nil, err
	}
	candidate.key, candidate.used, candidate.readers = key, c.clock, 1
	return candidate, nil
}

func (c *Cache) Read(ctx context.Context, request Request, offset int64, limit int) ([]byte, int64, string, error) {
	stamp := c.stamp(request)
	if request.Expected != "" && request.Expected != stamp {
		return nil, 0, "", faults.New(diagcatalog.Conflict, diagcatalog.ServerReadRevision)
	}
	slot, err := c.borrow(request)
	if err != nil {
		return nil, 0, "", err
	}
	defer func() { c.mu.Lock(); slot.readers--; c.mu.Unlock() }()
	page, total, err := slot.value.Read(ctx, request, offset, limit)
	if err != nil {
		return nil, 0, "", err
	}
	if c.stamp(request) != stamp {
		return nil, 0, "", faults.New(diagcatalog.Conflict, diagcatalog.ServerReadRevision)
	}
	return page, total, stamp, nil
}

func (c *Cache) stamp(request Request) string {
	body, _ := json.Marshal(struct {
		Epoch, Session, File, FileRevision string
		Source                             server.DetailSource
		Version                            int
	}{c.epoch, request.Session, request.File, request.FileRevision, request.Source, request.Version()})
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func (c *Cache) CloseSession(session string) error { return c.closeSlots(&session) }
func (c *Cache) Close() error                      { return c.closeSlots(nil) }

// Pin only affected slots under the pool mutex. Waiting for a document reader
// or closing its file must never block access to unrelated cached documents.
func (c *Cache) closeSlots(session *string) error {
	c.mu.Lock()
	var targets []*cacheSlot
	for _, slot := range c.slots {
		if session == nil || slot.key.session == *session {
			slot.readers++
			slot.closing++
			targets = append(targets, slot)
		}
	}
	c.mu.Unlock()
	var result error
	for _, slot := range targets {
		if session == nil {
			result = errors.Join(result, slot.value.Close())
		} else {
			result = errors.Join(result, slot.value.CloseSession(*session))
		}
	}
	c.mu.Lock()
	for _, slot := range targets {
		slot.readers--
		slot.closing--
	}
	c.mu.Unlock()
	return result
}
