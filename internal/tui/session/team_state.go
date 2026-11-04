// Purpose: Own persisted team-mode selection and connect its request snapshot to the display component.
// Pattern: Event-loop-owned feature state.
// Usage: Commands toggle team mode; each request snapshots it before workers start.
// Related: internal/tui/session/team_events.go, internal/tui/session/commands.go.
package session

import (
	teamview "github.com/keonho-kim/meca/internal/tui/chat/team"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type teamMode struct {
	enabled, active, skipInitial bool
	view                         teamview.State
}

func (t *teamMode) begin() {
	t.active, t.skipInitial = t.enabled, t.enabled
	if t.active {
		t.view.Begin()
	}
}
func (t *teamMode) finish(output *viewport.View) { t.view.Finish(output); t.active = false }
func (a *app) toggleTeam() {
	value := !a.team.enabled
	if a.opts.SaveTeamMode != nil {
		if err := a.opts.SaveTeamMode(a.ctx, value); err != nil {
			a.fail(err)
			return
		}
	}
	a.team.enabled = value
	text := "팀 모드 OFF"
	if value {
		text = "팀 모드 ON"
	}
	text = a.language.Text(text)
	if a.work.state.agent() {
		text += a.language.Text(" · 다음 요청부터 적용")
	}
	a.stopNotice = text
}
func (a *app) teamCaption() string {
	if !a.team.enabled && !a.team.active {
		return ""
	}
	return a.language.Format("팀 %d/8", a.team.view.Count())
}

func (a *app) footerModel() string {
	model := a.connection.selection.Model
	if caption := a.teamCaption(); caption != "" {
		model += " · " + caption
	}
	return model
}
