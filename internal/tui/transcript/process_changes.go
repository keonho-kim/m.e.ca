// Purpose: Present compact known-file changes observed after a shell command.
// Pattern: Pure presentation projection.
// Usage: Command result formatting consumes actual operation changes, never command text guesses.
// Related: internal/tui/transcript/details.go, internal/agent/run/command_files.go.
package transcript

import (
	"github.com/keonho-kim/v-code/internal/agent/run"
)

func (f Formatter) processChanges(event run.Event) string {
	if event.Operation == nil {
		return ""
	}
	changes := event.Operation.Changes
	text := ""
	for i, change := range changes {
		if i >= 12 {
			text += indentTool(f.Language.Format("그 외 관측된 변경 %d개 · 원본 기록에 보존", len(changes)-i))
			break
		}
		label := "변경 확인: "
		if change.AfterDigest == "missing" {
			label = "삭제 확인: "
		}
		text += indentTool(f.Language.Text(label) + f.Redact(change.Path))
	}
	return text
}
