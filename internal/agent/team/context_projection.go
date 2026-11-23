// Purpose: Avoid repeating public bodies already present in one peer's uncompacted conversation.
// Pattern: Peer-owned projection cached by immutable snapshot and coverage mask.
// Usage: The team runner supplies current history when assembling each model request.
// Related: internal/agent/team/work_context.go, internal/agent/team/contribution/context.go.
package team

import (
	"slices"

	model "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/contribution"
)

func (v *workProjection) fullOverview() string {
	v.renderOnce.Do(func() {
		if v.base != nil {
			v.text = v.render(nil)
		}
	})
	return v.text
}

func (v *workProjection) render(covered []bool) string {
	var coveredScopes, coveredDiscussion []bool
	if covered != nil {
		coveredScopes, coveredDiscussion = covered[:len(v.scopes)], covered[len(v.scopes):]
	}
	// Size the output from stable scopes without formatting a second copy.
	size := len(v.base.roster.text) + 128
	for i, cost := range v.base.sizes {
		if !coveredAt(coveredScopes, i) {
			size += cost
		}
	}
	for _, status := range v.statuses {
		size += len(status.from) + len(status.text) + 32
	}
	for i, item := range v.discussions {
		if !coveredAt(coveredDiscussion, i) {
			size += len(item.from) + len(item.text) + 64
		}
	}
	var out overviewBuffer
	out.body.Grow(min(size, teamOverviewBytes))
	out.WriteString(teamOverviewTitle)
	out.add(v.base.roster)
	writeCommandStatuses(&out, v.statuses)
	writeDiscussions(&out, v.discussions, coveredDiscussion)
	out.WriteString("\nCurrent contributions:\n")
	for i, scope := range v.scopes {
		if out.shortened {
			break
		}
		if !coveredAt(coveredScopes, i) {
			contribution.WriteContext(&out, scope)
		}
	}
	return out.finish()
}

func coveredAt(covered []bool, index int) bool {
	return index < len(covered) && covered[index]
}

func (p *Peer) releaseOverview() {
	p.receivedView, p.decisionView = decisionSnapshot{}, decisionSnapshot{}
	p.contextView, p.contextText = nil, ""
}

func (p *Peer) ContextObservations(messages []model.Message) string {
	snapshot := p.decisionView
	if snapshot.view == nil {
		snapshot = p.room.workSnapshot()
	}
	view := snapshot.view
	count := len(view.scopes) + len(view.discussions)
	if cap(p.coverageScratch) < count {
		p.coverageScratch = make([]bool, count)
	}
	covered := p.coverageScratch[:count]
	clear(covered)
	anyCovered := false
	for _, m := range messages {
		sequence, _ := channel.Sequence(m)

		if sequence == 0 {
			continue
		}
		if i, ok := view.sources[sequence]; ok {
			covered[i] = true
			anyCovered = true
		}
	}
	if !anyCovered {
		p.contextView, p.contextText = nil, ""
		overview := view.fullOverview()
		if p.idleDirective != "" {
			return overview + "\n" + p.idleDirective
		}
		return overview
	}
	if p.contextView != view || !slices.Equal(p.contextCovered, covered) {
		p.contextView = view
		p.contextCovered = append(p.contextCovered[:0], covered...)
		p.contextText = view.render(covered)
	}
	if p.idleDirective != "" {
		return p.contextText + "\n" + p.idleDirective
	}
	return p.contextText
}
