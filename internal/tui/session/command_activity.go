// Purpose: Refresh command snapshots without overwriting pending tools or user notices.
// Pattern: Event-loop projection of synchronized runtime observations.
// Usage: updateElapsed reads command metadata without adding workers or history records.
// Related: internal/tui/session/elapsed.go, internal/tui/session/tool_activity.go.
package session

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (a *app) updateCommandActivity() {
	if a.team.active {
		a.updateTeamCommands()
		return
	}
	commands, ok := a.opts.Tools.(toolapi.CommandExecutor)
	if !ok || a.work.state.stopping() {
		a.chat.toolActivity.dirty = a.chat.toolActivity.dirty || len(a.chat.toolActivity.commands) > 0
		a.chat.toolActivity.commands = nil
		return
	}
	a.chat.toolActivity.commands = commands.CommandActivity()
	a.chat.toolActivity.dirty = true
}
