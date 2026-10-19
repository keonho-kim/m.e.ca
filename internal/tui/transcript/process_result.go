// Purpose: Present process streams and termination status without protocol metadata.
// Pattern: Pure presentation projection.
// Usage: Tool result dispatch formats execute_command output.
// Related: internal/tui/transcript/details.go, internal/tools/local/process/run_unix.go.
package transcript

import (
	"encoding/json"
	"strings"
)

func (f Formatter) processOutput(fields map[string]json.RawMessage) string {
	text := ""
	for _, key := range []string{"Stdout", "Stderr"} {
		var value string
		if json.Unmarshal(fields[key], &value) == nil && value != "" {
			if strings.Contains(value, "\n") {
				text += indentTool(f.Language.Text(strings.ToLower(key))) + indentTool(f.Redact(value))
			} else {
				text += indentTool(f.Language.Text(strings.ToLower(key)) + ": " + f.Redact(value))
			}
		}
	}
	for _, key := range []string{"StdoutPath", "StderrPath"} {
		var path string
		if json.Unmarshal(fields[key], &path) == nil && path != "" {
			text += indentTool(f.Language.Text("captured log: ") + f.Redact(path))
		}
	}
	var code int
	if json.Unmarshal(fields["ExitCode"], &code) == nil {
		text += indentTool(f.Language.Text("exit: ") + string(fields["ExitCode"]))
	}
	var status string
	if json.Unmarshal(fields["Status"], &status) == nil && status != "exited" && status != "" {
		text += indentTool(f.Language.Text("status: ") + f.Language.Text(f.Redact(status)))
	}
	return text
}
