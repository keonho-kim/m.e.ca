// Purpose: Own the common five-second idle threshold and one-second heartbeat cadence.
// Pattern: Connection-local deadline state.
// Usage: SSE writers and native streaming subscriptions reset it only for application data.
// Related: internal/serverapi/eventstream/writer.go, internal/serverapi/watch.go.
package eventstream

import (
	"context"
	"time"
)

type Idle struct{ next time.Time }

func NewIdle() Idle        { return Idle{next: time.Now().Add(5 * time.Second)} }
func (i *Idle) Activity()  { i.next = time.Now().Add(5 * time.Second) }
func (i *Idle) Heartbeat() { i.next = time.Now().Add(time.Second) }
func (i *Idle) Due() bool  { return !time.Now().Before(i.next) }
func (i *Idle) PollContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, i.next)
}
