// Purpose: Distinguish textual tool invocation attempts from user-facing work reports.
// Pattern: Pure response boundary.
// Usage: The run owner rejects protocol markup as a final report; it is never executed.
// Related: internal/agent/run/finish_response.go, internal/agent/run/report.go.
package response

import (
	"encoding/json"
	"strings"
)

func ToolMarkup(text string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "<function=") && (strings.Contains(lower, "<tool_call>") || strings.Contains(lower, "<parameter=")) {
		return true
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(strings.TrimSpace(text)), &fields) == nil {
		if _, ok := fields["tool"]; ok {
			return true
		}
		if _, ok := fields["tool_calls"]; ok {
			return true
		}
	}
	return false
}
