// Purpose: Present literal process output and shared localized outcomes without exit-code metadata.
// Pattern: Pure presentation projection.
// Usage: Expanded and collapsed execute_command results share outcome labels and diagnostics.
// Related: internal/tui/transcript/details.go, internal/tui/transcript/tool_summary.go, internal/tools/local/process/run_unix.go.
package transcript

import (
	"encoding/json"
	"strings"

	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
)

func (f Formatter) processOutput(event run.Event, fields map[string]json.RawMessage) toolOutput {
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
	status := processStatus(event.Kind, fields)
	if status != "" {
		text += indentTool(f.Language.Text(status))
		if diagnostic := f.processDiagnostic(event); diagnostic != "" {
			text += indentTool(f.Language.Text("Failed: ") + diagnostic)
		}
	}
	return toolOutput{text: text, failureHandled: status != ""}
}

func processStatus(kind string, fields map[string]json.RawMessage) string {
	var status string
	if raw, ok := fields["Status"]; ok && json.Unmarshal(raw, &status) != nil {
		return ""
	}
	switch status {
	case "interrupted":
		return "Command interrupted"
	case "timeout":
		return "Command timed out"
	case "execution_failed":
		return "Command failed"
	case "", "exited":
		var code *int
		if json.Unmarshal(fields["ExitCode"], &code) != nil || code == nil {
			return ""
		}
		if *code == 0 && kind == "tool_succeeded" {
			return "Command succeeded"
		}
		return "Command failed"
	default:
		return ""
	}
}

func (f Formatter) processDiagnostic(event run.Event) string {
	if event.Operation == nil || event.Operation.Failure == nil {
		return ""
	}
	failure := event.Operation.Failure
	switch failure.Code {
	case tool.ProcessFailed, tool.Interrupted, tool.Timeout:
		// The outcome and streams already describe these process failures.
		return ""
	default:
		return f.Redact(failure.Detail)
	}
}

func (f Formatter) processSummary(event run.Event, fields map[string]json.RawMessage) string {
	if diagnostic := f.processDiagnostic(event); diagnostic != "" {
		return diagnostic
	}
	if event.Kind == "tool_succeeded" {
		return ""
	}
	var stderr string
	if json.Unmarshal(fields["Stderr"], &stderr) == nil {
		return f.Redact(strings.TrimSpace(stderr))
	}
	return ""
}
