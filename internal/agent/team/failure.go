// Purpose: Record shared observations before delivery without holding membership state locks.
// Pattern: Serialized durable event boundary.
// Usage: Public log and peer events stop the room on persistence or delivery failure.
// Related: internal/agent/team/inbox.go, internal/agent/team/worker.go.
package team

import (
	"context"
	"errors"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"time"
)

func (r *Room) fail(err error) {
	if err == nil {
		return
	}
	r.mu.Lock()
	r.failures = errors.Join(r.failures, err)
	r.mu.Unlock()
	r.cancel(err)
}
func (r *Room) deliver(event runrecord.Event, persist bool) error {
	r.eventMu.Lock()
	defer r.eventMu.Unlock()
	if persist && r.options.Record != nil {
		stored := event
		stored.MeasuredDuration = 0
		if event.Reference != nil {
			stored = runrecord.Event{Kind: "team_reference", Agent: event.Agent, Reference: event.Reference}
		}
		flush, cancel := context.WithTimeout(context.WithoutCancel(r.ctx), 5*time.Second)
		err := r.options.Record(flush, stored)
		cancel()
		if err != nil {
			r.fail(err)
			return err
		}
	}
	err := r.options.Emit(event)
	r.fail(err)
	return err
}
func (r *Room) notice(agent, text string) error {
	return r.deliver(runrecord.Event{Kind: "team_notice", Agent: agent, Text: text}, true)
}
