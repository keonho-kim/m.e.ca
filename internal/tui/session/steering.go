// Purpose: Queue direction changes and restore undelivered input after worker completion.
// Pattern: Event-loop input ownership.
// Usage: Enter during an active agent queues a message; runtime notices acknowledge delivery.
// Related: internal/tui/session/tasks.go, internal/agent/run/steering.go.
package session

import (
	"strings"

	requestinput "github.com/keonho-kim/meca/internal/commands/input"
)

func (a *app) steerInput(text string) {
	if a.work.state.stopping() {
		return
	}
	select {
	case a.compose.steering <- requestinput.Capture(text, a.selectedPaths()):
		select {
		case a.compose.steeringReady <- struct{}{}:
		default:
		}
		a.chat.output.Follow = true
		a.compose.steeringDrafts = append(a.compose.steeringDrafts, text)
		a.compose.editor.Set("")
		a.status = a.language.Text("현재 작업의 다음 판단에 반영합니다")
	default:
		a.status = a.language.Text("전달 대기 중입니다. 초안을 유지합니다")
	}
}

func (a *app) restoreSteering() {
	editor := &a.compose.editor
	if a.compose.queued.Edit != nil {
		editor = &a.compose.queued.Edit.Draft
	}
	if len(a.compose.steeringDrafts) > 0 {
		text := strings.Join(a.compose.steeringDrafts, "\n\n")
		if draft := editor.String(); draft != "" {
			text += "\n\n" + draft
		}
		editor.Set(text)
	}
	a.compose.steering, a.compose.steeringDrafts = nil, nil
	a.compose.steeringReady = nil
}
