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
				text += indentTool(strings.ToLower(key)) + indentTool(f.Redact(value))
			} else {
				text += indentTool(strings.ToLower(key) + ": " + f.Redact(value))
			}
		}
	}
	for _, key := range []string{"StdoutPath", "StderrPath"} {
		var path string
		if json.Unmarshal(fields[key], &path) == nil && path != "" {
			text += indentTool("captured log: " + f.Redact(path))
		}
	}
	var code int
	if json.Unmarshal(fields["ExitCode"], &code) == nil {
		text += indentTool("exit: " + string(fields["ExitCode"]))
	}
	var status string
	if json.Unmarshal(fields["Status"], &status) == nil && status != "exited" && status != "" {
		text += indentTool("status: " + f.Redact(status))
	}
	return text
}
