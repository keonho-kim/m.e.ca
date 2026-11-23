// Purpose: Share one coherent current-work projection across private model decisions.
// Pattern: Room-owned immutable snapshot invalidated by its exact state key.
// Usage: Inbox captures current membership and contributions outside the state lock.
// Related: internal/agent/team/work.go, internal/agent/team/contribution/context.go.
package team

import (
	"cmp"
	"fmt"
	"slices"
	"sync"

	"github.com/keonho-kim/meca/internal/agent/team/contribution"
	"github.com/keonho-kim/meca/internal/agent/team/source"
)

type workContextKey struct {
	membership uint64
	revision   uint64
	status     int64
	discussion int64
}
type workMember struct {
	name    string
	invited source.WorkDomain
}
type workBase struct {
	key     workContextKey
	members []workMember
	scopes  []contribution.Scope
	sizes   []int
	roster  overviewPart
}
type workProjection struct {
	base        *workBase
	key         workContextKey
	text        string
	renderOnce  sync.Once
	scopes      []contribution.Scope
	statuses    []commandStatus
	discussions []discussionPreview
	sources     map[int64]int
}
type decisionSnapshot struct {
	view    *workProjection
	through int64
}

// Caller holds room state; inactive work must not remain in either cache.
func (r *Room) invalidateWorkLocked() {
	r.membershipRevision++
	r.workView, r.workBase = nil, nil
}

// workKeyLocked captures membership and contributions at the same boundary.
func (r *Room) workKeyLocked() workContextKey {
	return workContextKey{membership: r.membershipRevision, revision: r.work.Revision(), status: r.latestStatus, discussion: r.latestDiscussion}
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
		members := make([]workMember, 0, len(r.active))
		for name, live := range r.active {
			if !live.closed && !live.leaving {
				members = append(members, workMember{name, live.invited})
			}
		}
		slices.SortFunc(members, func(a, b workMember) int { return cmp.Compare(a.name, b.name) })
		base = &workBase{key: baseKey, members: members, scopes: r.work.Snapshot()}
	}
	var statuses []commandStatus
	var discussions []discussionPreview
	for _, member := range base.members {
		live := r.active[member.name]
		if len(live.statuses) > 0 {
			for _, status := range live.statuses {
				statuses = append(statuses, status)
			}
		} else if live.status.sequence != 0 {
			statuses = append(statuses, live.status)
		}
		if live.discussion.sequence != 0 {
			discussions = append(discussions, live.discussion)
		}
	}
	slices.SortFunc(statuses, func(a, b commandStatus) int { return cmp.Compare(a.sequence, b.sequence) })
	r.mu.Unlock()
	if base.roster.text == "" {
		base.roster = base.renderRoster()
		base.sizes = make([]int, len(base.scopes))
		for i, scope := range base.scopes {
			base.sizes[i] = contribution.ContextSize(scope)
		}
	}
	view := &workProjection{key: key, base: base, scopes: base.scopes, statuses: statuses, discussions: discussions}
	if len(view.scopes)+len(view.discussions) > 0 {
		view.sources = make(map[int64]int, len(view.scopes)+len(view.discussions))
		for i, scope := range view.scopes {
			view.sources[scope.Sequence] = i
		}
		for i, note := range view.discussions {
			view.sources[note.sequence] = len(view.scopes) + i
		}
	}
	r.mu.Lock()
	// A slow formatter never replaces a newer snapshot made by another peer.
	if r.workKeyLocked() == key {
		r.workView = view
		r.workBase = base
	}
	r.mu.Unlock()
	return decisionSnapshot{view, through}
}

func (b *workBase) renderRoster() overviewPart {
	var out overviewBuffer
	out.WriteString("Active teammates: ")
	for i, member := range b.members {
		if out.shortened {
			break
		}
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(member.name)
	}
	out.WriteByte('\n')
	for _, member := range b.members {
		if out.shortened {
			break
		}
		if member.invited.Domain != "" {
			fmt.Fprintf(&out, "%s joined for domain %s\n", member.name, member.invited)
		}
	}
	if len(b.scopes) > 0 {
		out.WriteString("Current work references:\n")
	}
	for _, scope := range b.scopes {
		if out.shortened {
			break
		}
		fmt.Fprintf(&out, "%s: current work #%d, revision %d", scope.Member, scope.Sequence, scope.Revision)
		for _, domain := range scope.Domains {
			if out.shortened {
				break
			}
			fmt.Fprintf(&out, " · %s", domain)
		}
		out.WriteByte('\n')
	}
	return out.part()
}
