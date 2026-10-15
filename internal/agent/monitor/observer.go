// Purpose: Persist ordered runtime facts on the existing execution worker.
// Pattern: Synchronous durable observer.
// Usage: Run owns New/Close; state is published only after durable persistence.
// Related: internal/agent/run/monitor.go, internal/storage/sqlite/monitor.go.
package monitor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

type Record struct {
	Sequence                          int64
	RunID, Kind, CallID, Tool, Detail string
}
type Persist func(context.Context, Record) error
type Observer struct {
	mu       sync.Mutex
	sequence int64
	persist  Persist
	closed   bool
}

func New(seed []Record, persist Persist) *Observer {
	o := &Observer{persist: persist}
	for _, r := range seed {
		o.sequence = max(o.sequence, r.Sequence)
	}
	return o
}
func (r Record) Check() error {
	if r.Sequence < 0 || r.Kind == "" || len(r.Kind) > 64 || len(r.RunID) > 128 || len(r.CallID) > 128 || len(r.Tool) > 128 || len(r.Detail) > 1200 || !utf8.ValidString(r.Detail) {
		return fmt.Errorf("invalid Monitor record")
	}
	return nil
}
func Detail(text string) string {
	if len(text) <= 1024 {
		return text
	}
	end := 1024
	for !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + " [excerpt; see original event]"
}
func (o *Observer) Observe(ctx context.Context, r Record) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if o.closed {
		return fmt.Errorf("Monitor is closed")
	}
	r.Sequence = o.sequence + 1
	r.Detail = Detail(strings.TrimSpace(r.Detail))
	if err := r.Check(); err != nil {
		return err
	}
	if o.persist == nil {
		return fmt.Errorf("Monitor persistence required")
	}
	if err := o.persist(ctx, r); err != nil {
		return err
	}
	o.sequence = r.Sequence
	return nil
}
func (o *Observer) Close() { o.mu.Lock(); defer o.mu.Unlock(); o.closed = true }
