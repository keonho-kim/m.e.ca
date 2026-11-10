// Purpose: Project task-scoped subscriptions without mixing concurrent drafts or approvals.
// Pattern: Pure notification filter at the transport boundary.
// Usage: SSE, WebSocket and RPC apply taskSender before writing notifications.
// Related: internal/serverapi/watch.go, internal/serverapi/http.go.
package serverapi

import (
	"slices"

	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func taskSender(id string, send func(*v1.Notification) error) func(*v1.Notification) error {
	if id == "" {
		return send
	}
	return func(n *v1.Notification) error {
		if e := n.GetEvent(); e != nil && e.Kind != "heartbeat" && e.TaskId != id {
			return nil
		}
		if state := n.GetSnapshot(); state != nil {
			state.Tasks = slices.DeleteFunc(state.Tasks, func(t *v1.Task) bool { return t.Id != id })
			state.Executions = slices.DeleteFunc(state.Executions, func(e *v1.Execution) bool { return e.TaskId != id })
			state.Approvals = slices.DeleteFunc(state.Approvals, func(a *v1.Approval) bool { return a.TaskId != id })
			state.Drafts = slices.DeleteFunc(state.Drafts, func(d *v1.Draft) bool { return d.TaskId != id })
			if state.GetTask().GetId() != id {
				state.Task = nil
			}
			if state.GetExecution().GetTaskId() != id {
				state.Execution = nil
			}
		}
		return send(n)
	}
}
