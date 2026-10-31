// Purpose: Present tool targets and commands without repeating file bodies.
// Pattern: Pure presentation projection.
// Usage: Formatter.Tool renders parameters without repeating source bodies.
// Related: internal/tui/transcript/tools.go, internal/tui/transcript/file_results.go.
package transcript

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
)

func (f Formatter) arguments(call tool.Call) string {
	args, err := tool.DisplayArgs(call.Arguments)
	if err != nil {
		return indentTool(f.Language.Text("인자 형식 확인 필요"))
	}
	var text strings.Builder
	for _, field := range []struct{ value, label string }{{args.Path, "경로"}, {args.Command, "명령"}, {args.CommandID, "Command ID"}} {
		if field.value != "" {
			text.WriteString(indentTool(f.Language.Text(field.label) + ": " + f.Redact(field.value)))
		}
	}
	if args.WaitSeconds != nil {
		text.WriteString(indentTool(f.Language.Text("Wait seconds") + ": " + fmt.Sprint(*args.WaitSeconds)))
	}
	for _, path := range args.Paths {
		text.WriteString(indentTool(f.Language.Text("경로") + ": " + f.Redact(path)))
	}
	return text.String()
}
