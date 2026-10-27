// Purpose: Refresh live command information on the terminal's existing elapsed ticker.
// Pattern: Event-loop projection of synchronized runtime observations.
// Usage: updateElapsed reads command metadata without adding workers or history records.
// Related: internal/tui/session/elapsed.go, internal/tui/transcript/commands.go.
package session

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func (a *app) updateCommandActivity() {
	commands, ok := a.opts.Tools.(toolapi.CommandExecutor)
	if !ok {
		return
	}
	text := (transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).CommandActivity(commands.CommandActivity())
	if text != "" {
		a.status = text
		a.commandStatus = true
	} else if a.commandStatus {
		a.status = ""
		a.commandStatus = false
	}
}
