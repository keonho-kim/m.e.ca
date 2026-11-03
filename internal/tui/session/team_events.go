// Purpose: Route team application effects while the chat component owns stream and tool projection.
// Pattern: Event-loop application adapter.
// Usage: receiveRuntime dispatches attributed team events only on the terminal loop.
// Related: internal/tui/session/team_state.go, internal/tui/transcript/team.go.
package session

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func (a *app) receiveTeam(event *runrecord.Event) {
	if event.Agent == "user" {
		if a.team.skipInitial {
			a.team.skipInitial = false
			return
		}
		if len(a.compose.steeringDrafts) > 0 {
			a.compose.steeringDrafts[0] = ""
			a.compose.steeringDrafts = a.compose.steeringDrafts[1:]
		}
		a.chat.output.Append("> "+event.Text+"\n\n", false)
		a.compose.rememberInput(event.Text)
		return
	}
	f := transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact, RedactionVersion: a.opts.Vault.RedactionVersion}
	a.team.view.Apply(*event, f, &a.chat.output)
	switch event.Kind {
	case "compaction_started":
		a.startCompression()
	case "compaction_finished", "compaction", "team_notice":
		if !a.team.view.Compressing() {
			a.stopCompression()
		}
	case "usage_progress", "usage":
		if event.Kind == "usage" && event.Usage != nil {
			a.chat.tokenTotal = a.chat.tokenTotal.Add(*event.Usage)
		}
		current, rate := a.team.view.CurrentUsage()
		a.chat.tokenCurrent = current
		if rate != nil {
			a.chat.tokensPerSecond = rate
		}
	case "retry":
		a.status = event.Agent + " · " + event.Text
	case "image_input_unavailable":
		a.connection.rejectImages()
	}
}
func (a *app) updateTeamCommands() {
	if a.opts.Team == nil {
		return
	}
	f := transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact, RedactionVersion: a.opts.Vault.RedactionVersion}
	for agent, commands := range a.opts.Team.CommandActivity() {
		a.team.view.Commands(agent, commands, f, &a.chat.output)
	}
}
