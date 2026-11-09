// Purpose: Bound aggregate retained and projected model context across simultaneous executions.
// Pattern: Process-owned byte ledger with execution-owned reservations.
// Usage: Runtime reserves before hydration/projection and releases on every execution exit.
// Related: internal/agent/run/context_memory.go, internal/storage/sqlite/context.go.
package processmemory

import (
	"sync"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const ContextBytes int64 = 1 << 30

var contextBytes struct {
	sync.Mutex
	used int64
}

type ContextLease struct {
	bytes  int64
	closed bool
}

func NewContextLease() *ContextLease { return &ContextLease{} }

func (l *ContextLease) Resize(bytes int64) error {
	contextBytes.Lock()
	defer contextBytes.Unlock()
	if l.closed || bytes < 0 || bytes > ContextBytes || bytes-l.bytes > ContextBytes-contextBytes.used {
		return faults.New(diagcatalog.ResourceExhausted, diagcatalog.ContextMemoryLimit)
	}
	contextBytes.used += bytes - l.bytes
	l.bytes = bytes
	return nil
}

func (l *ContextLease) Close() {
	contextBytes.Lock()
	defer contextBytes.Unlock()
	if l.closed {
		return
	}
	contextBytes.used -= l.bytes
	l.bytes = 0
	l.closed = true
}
