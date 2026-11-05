// Purpose: Deliver committed public events in order after releasing the durable publication lock.
// Pattern: Joined caller-drained bounded queue.
// Usage: Publication owners finish through endPublication; no additional background worker is created.
// Related: internal/agent/team/publication.go, internal/agent/team/failure.go.
package team

import runrecord "github.com/keonho-kim/meca/internal/agent/record"

func (r *Room) queuePublic(event runrecord.Event) {
	if r.pendingPublic == nil {
		r.pendingPublic = make([]runrecord.Event, 0, MaxActive+1)
	}
	if len(r.pendingPublic) == cap(r.pendingPublic) && r.publicHead > 0 {
		count := copy(r.pendingPublic, r.pendingPublic[r.publicHead:])
		clear(r.pendingPublic[count:])
		r.pendingPublic = r.pendingPublic[:count]
		r.publicHead = 0
	}
	r.pendingPublic = append(r.pendingPublic, event)
}

// The caller still owns logMu; all durable and visible state is already committed.
func (r *Room) endPublication() error {
	r.logMu.Unlock()
	r.publicEmitMu.Lock()
	defer r.publicEmitMu.Unlock()
	for {
		r.logMu.Lock()
		if r.publicHead == len(r.pendingPublic) {
			r.logMu.Unlock()
			return nil
		}
		event := r.pendingPublic[r.publicHead]
		r.pendingPublic[r.publicHead] = runrecord.Event{}
		r.publicHead++
		if r.publicHead == len(r.pendingPublic) {
			r.pendingPublic = r.pendingPublic[:0]
			r.publicHead = 0
		}
		r.logMu.Unlock()
		if err := r.deliver(event, false); err != nil {
			return err
		}
	}
}
