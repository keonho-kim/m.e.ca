// Purpose: Avoid repeating public bodies already present in one peer's uncompacted conversation.
// Pattern: Peer-owned projection cached by immutable snapshot and coverage mask.
// Usage: The team runner supplies current history when assembling each model request.
// Related: internal/agent/team/work_context.go, internal/agent/team/contribution/context.go.
package team

import (
	"encoding/json"
	"fmt"
	model "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/contribution"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
)

func (v *workProjection) render(covered uint16) string {
	var out strings.Builder
	fmt.Fprintf(&out, "Active teammates: %s\nAvailable places: %d\nCurrent contributions:\n", strings.Join(v.key.names[:v.key.count], ", "), v.key.available)
	for i, scope := range v.scopes {
		contribution.WriteContext(&out, scope, covered&(1<<i) != 0)
	}
	writeCommandStatuses(&out, v.statuses[:v.key.count])
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
	var calls []tool.Call
	for _, m := range messages {
		if m.Role == "assistant" && len(m.Calls) > 0 {
			calls = m.Calls
		}
		sequence, _ := channel.Sequence(m.ID)
		if sequence == 0 && m.Role == "tool" && (m.ToolName == "discuss" || m.ToolName == "set_work") {
			paired := false
			for _, call := range calls {
				if call.ID == m.ToolCallID && call.Name == m.ToolName {
					paired = true
					break
				}
			}
			if !paired {
				continue
			}
			var outcome struct{ Sequence int64 }
			if json.Unmarshal([]byte(m.Content), &outcome) == nil {
				sequence = outcome.Sequence
			}
		}
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
		if p.idleDirective != "" {
			return view.text + "\n" + p.idleDirective
		}
		return view.text
	}
	if p.contextView != view || p.contextMask != covered {
		p.contextView, p.contextMask, p.contextText = view, covered, view.render(covered)
	}
	if p.idleDirective != "" {
		return p.contextText + "\n" + p.idleDirective
	}
	return p.contextText
}
