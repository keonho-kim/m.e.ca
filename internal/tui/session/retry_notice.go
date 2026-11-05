// Purpose: Retire a transient retry notice only when its own source resumes useful output.
// Pattern: Event-loop-owned presentation state.
// Usage: Runtime admission shares this rule between single and team events.
// Related: internal/tui/session/runtime_events.go, internal/tui/session/team_events.go.
package session

import runrecord "github.com/keonho-kim/meca/internal/agent/record"

type retryNotice struct{ source, text string }

func (a *app) updateRetryNotice(event *runrecord.Event) {
	if event.Kind == "retry" {
		text := event.Text
		if event.Agent != "" {
			text = event.Agent + " · " + text
		}
		a.status = text
		a.chat.retry = retryNotice{source: event.Agent, text: text}
		return
	}
	notice := &a.chat.retry
	if notice.text == "" || notice.source != event.Agent {
		return
	}
	switch event.Kind {
	case "usage_progress", "usage":
		if event.Usage == nil || event.Usage.Output <= 0 {
			return
		}
	case "stream":
		if event.Text == "" && event.Thinking == "" {
			return
		}
	case "progress", "team_progress", "tool_started", "tool_succeeded", "tool_failed", "tool_interrupted", "command_finished", "denied":
	default:
		return
	}
	if a.status == notice.text {
		a.status = ""
	}
	*notice = retryNotice{}
}
