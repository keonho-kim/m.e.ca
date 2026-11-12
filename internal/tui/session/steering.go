// Purpose: Queue direction changes and restore undelivered input after worker completion.
// Pattern: Event-loop input ownership.
// Usage: Enter during an active agent queues a message; runtime notices acknowledge delivery.
// Related: internal/tui/session/tasks.go, internal/agent/run/steering.go.
package session

import (
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tui/composer/queue"
)

func (a *app) steerInput(text string) {
	if a.work.state.stopping() {
		return
	}
	request := requestinput.Capture(text, a.selectedPaths())
	bytes := request.TextBytes()
	for _, pending := range a.compose.steeringDrafts {
		bytes += pending.TextBytes()
	}
	if bytes > queue.MaxTextBytes {
		a.fail(fault.New(diag.ResourceExhausted, diag.ServerQueueBytes))
		return
	}
	select {
	case a.compose.steering <- request:
		select {
		case a.compose.steeringReady <- struct{}{}:
		default:
		}
		a.chat.output.Follow = true
		a.compose.steeringDrafts = append(a.compose.steeringDrafts, request)
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
	if !queue.RestoreText(editor, a.compose.steeringDrafts) {
		a.compose.queued.Recover(a.compose.steeringDrafts)
	}
	a.compose.steering, a.compose.steeringDrafts = nil, nil
	a.compose.steeringReady = nil
}

func (r *requestState) acknowledgeSteering() {
	if len(r.steeringDrafts) > 0 {
		r.steeringDrafts[0] = requestinput.Request{}
		r.steeringDrafts = r.steeringDrafts[1:]
	}
}
