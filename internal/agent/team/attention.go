// Purpose: Target additional peer inference while retaining universal public delivery.
// Pattern: Deterministic current-scope notification policy.
// Usage: Ordered publication evaluates attention before replacing the source contribution.
// Related: internal/agent/team/publication.go, internal/agent/team/contribution/state.go.
package team

import (
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/contribution"
)

func (r *Room) needsAttentionLocked(value channel.Message, member string, work *contribution.Scope) bool {
	if !value.Attention(member) {
		return false
	}
	if value.Note != nil && value.Note.ReplyAuthor == member {
		return true
	}
	if value.Declaration && value.Note != nil && value.Note.Work != nil {
		if work != nil {
			if r.work.SameCoordination(*work) {
				return false
			}
			if r.work.ContractChanged(*work) {
				return true
			}
		}
		return r.work.Related(member, value.From, value.Note.Work.Paths)
	}
	if value.Observation {
		if len(value.ChangedPaths) == 0 {
			return true
		}
		return r.work.Related(member, "", value.ChangedPaths)
	}
	return true
}
