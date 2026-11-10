// Purpose: Apply one fold preference to live activity and completed tool results.
// Pattern: Event-loop presentation state transition.
// Usage: Ctrl+O and keyboard hints share the actual available fold targets.
// Related: internal/tui/session/tool_activity.go, internal/tui/session/keys.go, internal/tui/viewport/details.go.
package session

func (a *app) hasToolDetails() bool {
	return a.chat.output.HasDetails() || a.chat.toolActivity.hasDetails()
}

func (a *app) toggleToolDetails() {
	if !a.hasToolDetails() {
		a.stopNotice = a.language.Text("접거나 펼칠 도구 결과가 없습니다")
		return
	}
	a.chat.output.ToggleDetails()
	// Live activity is outside history, so its anchor needs the same reset as a
	// historical detail block when explicit folding changes the source rows.
	if tools := &a.chat.chatProjection.tools; tools.Text != "" && a.chat.output.AnchorID == tools.ID {
		a.chat.output.AnchorRow = 0
	}
	if a.chat.output.Collapsed {
		a.cancelToolDetail()
		a.stopNotice = a.language.Text("도구 결과 접음 · Ctrl+O로 펼치기")
	} else {
		a.stopNotice = a.language.Text("도구 결과 펼침 · PgUp/Dn으로 스크롤")
	}
}
