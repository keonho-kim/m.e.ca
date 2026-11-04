// Purpose: Share one coherent current-work projection across private model decisions.
// Pattern: Room-owned immutable snapshot invalidated by its exact state key.
// Usage: Inbox captures current membership and contributions outside the state lock.
// Related: internal/agent/team/work.go, internal/agent/team/contribution/context.go.
package team

import (
	"slices"

	"github.com/keonho-kim/meca/internal/agent/team/contribution"
)

type workContextKey struct {
	names            [MaxActive]string
	count, available int
	revision         uint64
	status           int64
	discussion       int64
}
type workProjection struct {
	key         workContextKey
	text        string
	scopes      []contribution.Scope
	statuses    [MaxActive]commandStatus
	discussions [MaxActive]discussionPreview
}
type decisionSnapshot struct {
	view    *workProjection
	through int64
}

// workKeyLocked captures membership and contributions at the same boundary.
func (r *Room) workKeyLocked() workContextKey {
	key := workContextKey{available: MaxActive - r.countLocked(), revision: r.work.Revision(), status: r.latestStatus, discussion: r.latestDiscussion}
	for name, live := range r.active {
		if !live.closed && !live.leaving {
			key.names[key.count] = name
			key.count++
		}
	}
	slices.Sort(key.names[:key.count])
	return key
}

func (r *Room) workSnapshot() decisionSnapshot {
	r.mu.Lock()
	key := r.workKeyLocked()
	if cached := r.workView; cached != nil && cached.key == key {
		through := r.latestPublic
		r.mu.Unlock()
		return decisionSnapshot{cached, through}
	}
	r.mu.Unlock()
	// Concurrent peers share a single rebuild without blocking membership updates.
	r.workViewMu.Lock()
	defer r.workViewMu.Unlock()
	r.mu.Lock()
	key = r.workKeyLocked()
	if cached := r.workView; cached != nil && cached.key == key {
		through := r.latestPublic
		r.mu.Unlock()
		return decisionSnapshot{cached, through}
	}
	through := r.latestPublic
	snapshot := r.work.Snapshot()
	var statuses [MaxActive]commandStatus
	var discussions [MaxActive]discussionPreview
	for i, name := range key.names[:key.count] {
		statuses[i] = r.active[name].status
		discussions[i] = r.active[name].discussion
	}
	r.mu.Unlock()
	view := &workProjection{key: key, scopes: snapshot, statuses: statuses, discussions: discussions}
	view.text = view.render(0)
	r.mu.Lock()
	// A slow formatter never replaces a newer snapshot made by another peer.
	if r.workKeyLocked() == key {
		r.workView = view
	}
	r.mu.Unlock()
	return decisionSnapshot{view, through}
}
