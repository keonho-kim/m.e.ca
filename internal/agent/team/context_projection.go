// Purpose: Avoid repeating public bodies already present in one peer's uncompacted conversation.
// Pattern: Peer-owned projection cached by immutable snapshot and coverage mask.
// Usage: The team runner supplies current history when assembling each model request.
// Related: internal/agent/team/work_context.go, internal/agent/team/contribution/context.go.
package team

import (
	"strings"

	model "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func (v *workProjection) render(covered uint16) string {
	base := ""
	if v.base != nil {
		base = v.base.text
		if covered&((1<<MaxActive)-1) != 0 {
			base = v.base.render(covered)
		}
	}
	return v.overlay(base, covered)
}
func (v *workProjection) overlay(base string, covered uint16) string {
	var out strings.Builder
	out.WriteString(base)
	writeCommandStatuses(&out, v.statuses)
	values := v.discussions
	for i := range values {
		if covered&(1<<(MaxActive+i)) != 0 {
			values[i] = discussionPreview{}
		}
	}
	writeDiscussions(&out, values[:v.key.count])
	return out.String()
}

func (p *Peer) ContextObservations(messages []model.Message) string {
	snapshot := p.decisionView
	if snapshot.view == nil {
		snapshot = p.room.workSnapshot()
	}
	view := snapshot.view
	var covered uint16
	for _, m := range messages {
		sequence, _ := channel.Sequence(m)

		if sequence == 0 {
			continue
		}
		for i, scope := range view.scopes {
			if scope.Sequence == sequence {
				covered |= 1 << i
			}
		}
		for i, note := range view.discussions {
			if note.sequence == sequence {
				covered |= 1 << (MaxActive + i)
			}
		}
	}
	if covered == 0 {
		p.contextView, p.contextText = nil, ""
		p.contextBase, p.contextBaseText = nil, ""
		if p.idleDirective != "" {
			return view.text + "\n" + p.idleDirective
		}
		return view.text
	}
	if p.contextView != view || p.contextMask != covered {
		scopeMask := covered & ((1 << MaxActive) - 1)
		if view.base != nil && (p.contextBase != view.base || p.contextBaseMask != scopeMask) {
			p.contextBase, p.contextBaseMask = view.base, scopeMask
			p.contextBaseText = view.base.text
			if scopeMask != 0 {
				p.contextBaseText = view.base.render(scopeMask)
			}
		}
		p.contextView, p.contextMask, p.contextText = view, covered, view.overlay(p.contextBaseText, covered)
	}
	if p.idleDirective != "" {
		return p.contextText + "\n" + p.idleDirective
	}
	return p.contextText
}
