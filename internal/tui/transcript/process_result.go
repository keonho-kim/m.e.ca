// Purpose: Present literal process output and shared localized outcomes without exit-code metadata.
// Pattern: Pure presentation projection.
// Usage: Expanded and collapsed execute_command results share outcome labels and diagnostics.
// Related: internal/tui/transcript/details.go, internal/tui/transcript/tool_summary.go, internal/tools/local/process/run_unix.go.
package transcript

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	"encoding/json"
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (f Formatter) processOutput(event runrecord.Event, fields map[string]json.RawMessage) toolOutput {
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
	if commandOriginalUnavailable(event) {
		status = "Command output unavailable"
	}
	if status != "" {
		text += indentTool(f.Language.Text(status))
		if diagnostic := f.processDiagnostic(event); diagnostic != "" {
			text += indentTool(f.Language.Text("Failed: ") + diagnostic)
		}
	}
	return toolOutput{text: text, failureHandled: status != ""}
}

func processStatus(kind string, fields map[string]json.RawMessage) string {
	if raw, exists := fields["output_unavailable"]; exists {
		var unavailable bool
		if json.Unmarshal(raw, &unavailable) == nil && unavailable {
			return "Command output unavailable"
		}
	}
	var status string
	if raw, ok := fields["Status"]; ok && json.Unmarshal(raw, &status) != nil {
		return ""
	}
	switch status {
	case "finalizing":
		return "Finalizing command output"
	case "waiting":
		return "Command waiting"
	case "running":
		return "Command running"
	case diagcatalog.Interrupted:
		return "Command interrupted"
	case diagcatalog.Timeout:
		return "Command timed out"
	case diagcatalog.ExecutionFailed:
		return "Command failed"
	case "", "exited":
		var code *int
		if json.Unmarshal(fields["ExitCode"], &code) != nil || code == nil {
			return ""
		}
		if *code == 0 && (kind == "tool_succeeded" || kind == "command_finished") {
			return "Command succeeded"
		}
		return "Command failed"
	default:
		return ""
	}
}

func commandOriginalUnavailable(event runrecord.Event) bool {
	return event.Operation != nil && event.Operation.Failure != nil && diagcatalog.Semantic(event.Operation.Failure.Code) == diagcatalog.PersistenceFailed
}

func (f Formatter) processDiagnostic(event runrecord.Event) string {
	if event.Operation == nil || event.Operation.Failure == nil {
		return ""
	}
	failure := event.Operation.Failure
	switch failure.Code {
	case diagcatalog.ProcessFailed, diagcatalog.Interrupted, diagcatalog.Timeout:
		// The outcome and streams already describe these process failures.
		return ""
	default:
		return f.Redact(failure.Detail)
	}
}

func (f Formatter) processSummary(event runrecord.Event, fields map[string]json.RawMessage) string {
	if diagnostic := f.processDiagnostic(event); diagnostic != "" {
		return diagnostic
	}
	if event.Kind == "tool_succeeded" {
		return ""
	}
	if event.Operation != nil && event.Operation.Process != nil && event.Operation.Process.Diagnostic != "" {
		return f.Redact(strings.TrimSpace(event.Operation.Process.Diagnostic))
	}
	var stderr string
	if json.Unmarshal(fields["Stderr"], &stderr) == nil {
		return f.Redact(strings.TrimSpace(stderr))
	}
	var output struct{ Stderr string }
	if json.Unmarshal(event.Output, &output) == nil {
		return f.Redact(strings.TrimSpace(output.Stderr))
	}
	return ""
}
