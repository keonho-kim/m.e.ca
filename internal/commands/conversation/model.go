// Purpose: Own one prepared model connection through explicit conversation and execution borrows.
// Pattern: Reference-counted resource handles with idempotent release.
// Usage: Conversation replacement and request workers release their own handles after use.
// Related: internal/commands/server/model_transfer.go, internal/commands/server/execution.go.
package conversation

import (
	"sync"
	"sync/atomic"

	"github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type modelConnection struct {
	mu        sync.Mutex
	owners    int
	generator model.Generator
	close     func()
}

type Model struct {
	connection *modelConnection
	released   atomic.Bool
}

func NewModel(generator model.Generator, close func()) *Model {
	return &Model{connection: &modelConnection{owners: 1, generator: generator, close: close}}
}

func (m *Model) Borrow() (*Model, error) {
	if m == nil {
		return nil, faults.Errorf(diagcatalog.ConversationModelConnectionIsUnavailable)
	}
	c := m.connection
	c.mu.Lock()
	defer c.mu.Unlock()
	if m.released.Load() || c.owners == 0 {
		return nil, faults.Errorf(diagcatalog.ConversationModelConnectionWasReleased)
	}
	c.owners++
	return &Model{connection: c}, nil
}

func (m *Model) Generator() model.Generator {
	c := m.connection
	c.mu.Lock()
	defer c.mu.Unlock()
	if m.released.Load() {
		return nil
	}
	return c.generator
}

func (m *Model) Close() {
	if m == nil || !m.released.CompareAndSwap(false, true) {
		return
	}
	c := m.connection
	c.mu.Lock()
	c.owners--
	var close func()
	if c.owners == 0 {
		close, c.close = c.close, nil
		c.generator = nil
	}
	c.mu.Unlock()
	if close != nil {
		close()
	}
}
