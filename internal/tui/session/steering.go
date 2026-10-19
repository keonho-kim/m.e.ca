// Purpose: Queue direction changes and restore undelivered input after worker completion.
// Pattern: Event-loop input ownership.
// Usage: Enter during an active agent queues a message; runtime notices acknowledge delivery.
// Related: internal/tui/session/tasks.go, internal/agent/run/steering.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/commands"
	"strings"
)

func (a *app) steerInput() {
	command, err := commands.Parse(a.editor.String())
	if err != nil {
		a.fail(err)
		return
	}
	if command.Kind != commands.Message || a.stopRequested {
		return
	}
	select {
	case a.steering <- command.Argument:
		a.steeringDrafts = append(a.steeringDrafts, command.Argument)
		a.editor.Set("")
		a.status = "현재 작업의 다음 판단에 반영합니다"
	default:
		a.status = "전달 대기 중입니다. 초안을 유지합니다"
	}
}

func (a *app) restoreSteering() {
	if len(a.steeringDrafts) > 0 {
		text := strings.Join(a.steeringDrafts, "\n\n")
		if draft := a.editor.String(); draft != "" {
			text += "\n\n" + draft
		}
		a.editor.Set(text)
	}
	a.steering, a.steeringDrafts = nil, nil
}
