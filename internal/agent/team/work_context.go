// Purpose: Share one coherent current-work projection across private model decisions.
// Pattern: Room-owned immutable snapshot invalidated by its exact state key.
// Usage: Inbox captures current membership and contributions outside the state lock.
// Related: internal/agent/team/work.go, internal/agent/team/contribution/context.go.
package team

import (
	"cmp"
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
	discussion       int64
}
type workBase struct {
	key    workContextKey
	scopes []contribution.Scope
	text   string
}
type workProjection struct {
	base        *workBase
	key         workContextKey
	text        string
	scopes      []contribution.Scope
	statuses    []commandStatus
	discussions [MaxActive]discussionPreview
}
type decisionSnapshot struct {
	view    *workProjection
	through int64
}

// Caller holds room state; inactive work must not remain in either cache.
func (r *Room) invalidateWorkLocked() { r.workView, r.workBase = nil, nil }

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
	baseKey := key
	baseKey.status, baseKey.discussion = 0, 0
	base := r.workBase
	if base == nil || base.key != baseKey {
		base = &workBase{key: baseKey, scopes: r.work.Snapshot()}
	}
	var statuses []commandStatus
	var discussions [MaxActive]discussionPreview
	for i, name := range key.names[:key.count] {
		live := r.active[name]
		if len(live.statuses) > 0 {
			for _, status := range live.statuses {
				statuses = append(statuses, status)
			}
		} else if live.status.sequence != 0 {
			statuses = append(statuses, live.status)
		}
		discussions[i] = live.discussion
	}
	slices.SortFunc(statuses, func(a, b commandStatus) int { return cmp.Compare(a.sequence, b.sequence) })
	r.mu.Unlock()
	if base.text == "" {
		base.text = base.render(0)
	}
	view := &workProjection{key: key, base: base, scopes: base.scopes, statuses: statuses, discussions: discussions}
	view.text = view.render(0)
	r.mu.Lock()
	// A slow formatter never replaces a newer snapshot made by another peer.
	if r.workKeyLocked() == key {
		r.workView = view
		r.workBase = base
	}
	r.mu.Unlock()
	return decisionSnapshot{view, through}
}

func (b *workBase) render(covered uint16) string {
	var out strings.Builder
	fmt.Fprintf(&out, "Active teammates: %s\nAvailable places: %d\nCurrent contributions:\n", strings.Join(b.key.names[:b.key.count], ", "), b.key.available)
	for i, scope := range b.scopes {
		contribution.WriteContext(&out, scope, covered&(1<<i) != 0)
	}
	return out.String()
}
