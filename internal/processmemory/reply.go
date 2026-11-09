// Purpose: Bound aggregate model reply growth without waiting while retaining partial replies.
// Pattern: Request-owned lease over a process-wide nonblocking byte budget.
// Usage: Each execution resizes its lease before appending model output and closes it on exit.
// Related: internal/agent/run/generate.go, internal/agent/run/start.go, internal/processmemory/transient.go.
package processmemory

import (
	"sync"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const ReplyBytes int64 = 256 << 20

var replies struct {
	mu   sync.Mutex
	used int64
}

// ReplyLease cannot wait while retaining an incomplete response: concurrent
// growers would otherwise wait on one another. Exhaustion stops that reply.
type ReplyLease struct {
	bytes  int64
	closed bool
}

func NewReplyLease() *ReplyLease { return &ReplyLease{} }
func (l *ReplyLease) Resize(bytes int64) error {
	replies.mu.Lock()
	defer replies.mu.Unlock()
	if l.closed || bytes < 0 || bytes > ReplyBytes || bytes-l.bytes > ReplyBytes-replies.used {
		return faults.New(diagcatalog.ResourceExhausted, diagcatalog.ReplyMemoryLimit)
	}
	replies.used += bytes - l.bytes
	l.bytes = bytes
	return nil
}
func (l *ReplyLease) Close() {
	replies.mu.Lock()
	defer replies.mu.Unlock()
	if !l.closed {
		replies.used -= l.bytes
		l.bytes = 0
		l.closed = true
	}
}
