// Purpose: Suspend file browsing while approval owns foreground input and restore it afterwards.
// Pattern: Explicit foreground state transitions.
// Usage: Runtime approval notices enter here; decisions and worker cleanup leave here.
// Related: internal/tui/session/events.go, internal/tui/session/menu_keys.go, internal/tui/session/worker_result.go.
package session

import toolapi "github.com/keonho-kim/meca/internal/agent/tool"

func (a *app) openApproval(request *toolapi.Approval) {
	a.approvalReturn = "chat"
	if a.page == "files" {
		a.approvalReturn = "files"
		a.stopFileWatch()
	}
	a.clearSelection()
	a.menuInput.Set("")
	a.proposal, a.page = request, "approval"
	a.choices, a.choice, a.previewOffset = []string{"Approve", "Reject"}, 0, 0
}

func (a *app) closeApproval() {
	a.page = "chat"
	if a.approvalReturn == "files" {
		a.page = "files"
	}
	a.proposal, a.approvalReturn = nil, ""
	a.finishFileSubmission()
}
