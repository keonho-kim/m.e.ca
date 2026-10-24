// Purpose: Present tool targets and commands without repeating file bodies.
// Pattern: Pure presentation projection.
// Usage: Formatter.Tool renders parameters without repeating source bodies.
// Related: internal/tui/transcript/tools.go, internal/tui/transcript/file_results.go.
package transcript

import (
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
)

func (f Formatter) arguments(call tool.Call) string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(call.Arguments, &fields) != nil {
		return indentTool(f.Language.Text("인자 형식 확인 필요"))
	}
	var text strings.Builder
	for _, field := range []struct{ key, label string }{{"path", "경로"}, {"destination", "이동할 경로"}, {"command", "명령"}} {
		var value string
		if json.Unmarshal(fields[field.key], &value) == nil {
			text.WriteString(indentTool(f.Language.Text(field.label) + ": " + f.Redact(value)))
		}
	}
	var paths []string
	if json.Unmarshal(fields["paths"], &paths) == nil {
		for _, path := range paths {
			text.WriteString(indentTool(f.Language.Text("경로") + ": " + f.Redact(path)))
		}
	}
	return text.String()
}
