// Purpose: Show rejected review findings and locations alongside the tool outcome.
// Pattern: Presentation projection.
// Usage: Live and restored Conclave results share this redacted formatter.
// Related: internal/tui/session/events.go, internal/tui/transcript/events.go, internal/agent/run/conclave.go.
package transcript

import (
	"fmt"
	"strings"

	"github.com/keonho-kim/v-code/internal/agent/run"
)

const ReviewRejectedMessage = "수정 심사를 통과하지 못하였습니다. 다시 작업을 진행합니다."

func (f Formatter) Conclave(event run.Event) string {
	if event.Call == nil || event.Conclave == nil || event.Conclave.Passed {
		return ""
	}
	var text strings.Builder
	text.WriteString("\n  " + f.Redact(event.Call.Name) + " · Conclave Fail\n\n")
	for _, finding := range event.Conclave.Verdict.Findings {
		location := finding.Path
		if finding.Line > 0 {
			location += fmt.Sprintf(":%d", finding.Line)
		}
		switch finding.Side {
		case "before":
			location += " (변경 전)"
		case "after":
			location += " (변경 후)"
		case "request":
			location += " (요청)"
		}
		text.WriteString(indentTool(f.Redact("- " + location + ": " + finding.Reason)))
	}
	if event.Conclave.Error != "" {
		text.WriteString(indentTool("심사 오류: " + f.Redact(event.Conclave.Error)))
	}
	text.WriteString("\n" + ReviewRejectedMessage + "\n")
	return text.String()
}
