// Purpose: Charge selected original reads to the shared payload budget as they grow.
// Pattern: Caller-owned nonblocking byte reservation.
// Usage: API originals reserve before hydration and retain only their output charge during delivery.
// Related: internal/processmemory/transient.go, internal/serverapi/detail/cache.go.
package processmemory

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type PayloadLease struct {
	bytes  int64
	closed bool
}

func NewPayloadLease() *PayloadLease { return &PayloadLease{} }

// Resize never waits while an original is retained. Waiting growers could
// otherwise keep one another, or a journal writer, from releasing capacity.
func (l *PayloadLease) Resize(bytes int) error {
	payloads.mu.Lock()
	defer payloads.mu.Unlock()
	next := int64(bytes)
	if l.closed || next < 0 || next > payloads.maximum || next-l.bytes > payloads.maximum-payloads.used {
		return faults.New(diagcatalog.ResourceExhausted, diagcatalog.PayloadMemoryLimit)
	}
	payloads.used += next - l.bytes
	if next < l.bytes {
		payloads.wake()
	}
	l.bytes = next
	return nil
}

func (l *PayloadLease) Close() {
	payloads.mu.Lock()
	defer payloads.mu.Unlock()
	if !l.closed {
		payloads.used -= l.bytes
		l.bytes, l.closed = 0, true
		payloads.wake()
	}
}
