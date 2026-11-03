// Purpose: Serialize real user approval and preserve refusals across peer executors.
// Pattern: Request-owned approval boundary.
// Usage: Every private peer delegates prepared approvals to the same request instance.
// Related: internal/commands/teamrequest/request.go, internal/tools/team/workspace/denial.go.
package teamrequest

import (
	"context"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (r *request) approve(ctx context.Context, peer string, proposal toolapi.Approval) (bool, error) {
	proposal.Name = peer + " · " + proposal.Name
	r.approval.Lock()
	defer r.approval.Unlock()
	approved, err := r.approver(ctx, proposal)
	if err == nil && !approved {
		r.access.Deny(proposal)
	}
	return approved, err
}
