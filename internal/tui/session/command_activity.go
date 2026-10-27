// Purpose: Refresh command snapshots without overwriting pending tools or user notices.
// Pattern: Event-loop projection of synchronized runtime observations.
// Usage: updateElapsed reads command metadata without adding workers or history records.
// Related: internal/tui/session/elapsed.go, internal/tui/session/tool_activity.go.
package session

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (a *app) updateCommandActivity() {
	commands, ok := a.opts.Tools.(toolapi.CommandExecutor)
	if !ok || a.worker.stopping() {
		a.toolActivity.dirty = a.toolActivity.dirty || len(a.toolActivity.commands) > 0
		a.toolActivity.commands = nil
		return
	}
	a.toolActivity.commands = commands.CommandActivity()
	a.toolActivity.dirty = true
}
