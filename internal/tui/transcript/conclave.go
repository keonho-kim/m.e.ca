// Purpose: Show only the tool's review outcome and a runtime-authored regeneration notice.
// Pattern: Presentation projection.
// Usage: Live and restored Conclave rejections share this formatter; votes and reasons remain internal.
// Related: internal/tui/session/events.go, internal/tui/transcript/events.go, internal/agent/run/conclave.go.
package transcript

import "github.com/keonho-kim/v-code/internal/agent/run"

const ReviewRejectedMessage = "수정 심사를 통과하지 못하였습니다. 다시 작업을 진행합니다."

func (f Formatter) Conclave(event run.Event) string {
	if event.Call == nil || event.Conclave == nil || event.Conclave.Passed {
		return ""
	}
	return "\n  " + f.Redact(event.Call.Name) + " · Conclave Fail\n\n" + ReviewRejectedMessage + "\n"
}
