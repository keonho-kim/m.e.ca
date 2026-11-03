// Purpose: Share one coherent current-work projection across private model decisions.
// Pattern: Room-owned immutable snapshot invalidated by its exact state key.
// Usage: Peer.WorkContext formats changed membership or contributions outside the state lock.
// Related: internal/agent/team/work.go, internal/agent/team/contribution/context.go.
package team

import (
	"fmt"
	"slices"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/team/contribution"
)

type workContextKey struct {
	names            [MaxActive]string
	count, available int
	revision         uint64
	status           int64
}
type workProjection struct {
	key  workContextKey
	text string
}

// workKeyLocked captures membership and contributions at the same boundary.
func (r *Room) workKeyLocked() workContextKey {
	key := workContextKey{available: MaxActive - r.countLocked(), revision: r.work.Revision(), status: r.latestStatus}
	for name, live := range r.active {
		if !live.closed && !live.leaving {
			key.names[key.count] = name
			key.count++
		}
	}
	slices.Sort(key.names[:key.count])
	return key
}

func (p *Peer) WorkContext() string {
	r := p.room
	r.mu.Lock()
	key := r.workKeyLocked()
	if cached := r.workView; cached.text != "" && cached.key == key {
		r.mu.Unlock()
		return cached.text
	}
	r.mu.Unlock()
	// Concurrent peers share a single rebuild without blocking membership updates.
	r.workViewMu.Lock()
	defer r.workViewMu.Unlock()
	r.mu.Lock()
	key = r.workKeyLocked()
	if cached := r.workView; cached.text != "" && cached.key == key {
		r.mu.Unlock()
		return cached.text
	}
	snapshot := r.work.Snapshot()
	var statuses [MaxActive]commandStatus
	for i, name := range key.names[:key.count] {
		statuses[i] = r.active[name].status
	}
	r.mu.Unlock()
	var out strings.Builder
	fmt.Fprintf(&out, "Active teammates: %s\nAvailable places: %d\nCurrent contributions:\n%s",
		strings.Join(key.names[:key.count], ", "), key.available, strings.TrimSpace(contribution.Context(snapshot)))
	writeCommandStatuses(&out, statuses[:key.count])
	text := out.String()
	r.mu.Lock()
	// A slow formatter never replaces a newer snapshot made by another peer.
	if r.workKeyLocked() == key {
		r.workView = workProjection{key: key, text: text}
	}
	r.mu.Unlock()
	return text
}
