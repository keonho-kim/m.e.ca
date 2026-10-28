// Purpose: Queue direction changes and restore undelivered input after worker completion.
// Pattern: Event-loop input ownership.
// Usage: Enter during an active agent queues a message; runtime notices acknowledge delivery.
// Related: internal/tui/session/tasks.go, internal/agent/run/steering.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/composer"
	"strings"
)

func (a *app) steerInput(text string) {
	if a.worker.stopping() {
		return
	}
	select {
	case a.steering <- composer.Capture(text, a.selectedPaths()):
		a.output.Follow = true
		a.steeringDrafts = append(a.steeringDrafts, text)
		a.editor.Set("")
		a.status = a.language.Text("현재 작업의 다음 판단에 반영합니다")
	default:
		a.status = a.language.Text("전달 대기 중입니다. 초안을 유지합니다")
	}
}

func (a *app) restoreSteering() {
	editor := &a.editor
	if a.queued.Edit != nil {
		editor = &a.queued.Edit.Draft
	}
	if len(a.steeringDrafts) > 0 {
		text := strings.Join(a.steeringDrafts, "\n\n")
		if draft := editor.String(); draft != "" {
			text += "\n\n" + draft
		}
		editor.Set(text)
	}
	a.steering, a.steeringDrafts = nil, nil
}
