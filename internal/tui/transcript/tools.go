// Purpose: Format tool parameters and actual results with indentation and literal-value redaction.
// Pattern: Presentation projection.
// Usage: Live and archived tool events use Formatter.Tool.
// Related: internal/agent/run/events.go, internal/tui/transcript/events.go.
package transcript

import (
	"bytes"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"sort"
	"strings"
)

func (f Formatter) Tool(event run.Event) string {
	if event.Call == nil {
		return ""
	}
	var text strings.Builder
	label := event.Call.Name
	if event.Conclave != nil {
		if event.Conclave.Passed {
			label += " · Conclave Pass"
		} else {
			label += " · Conclave Fail"
		}
	}
	text.WriteString("\n  " + label + "\n\n")
	text.WriteString(f.toolValues(event.Call.Arguments))
	if event.Kind == "tool_started" {
		text.WriteString("\n    Running…\n")
	} else {
		text.WriteString("\n    Output\n")
		if len(event.Output) > 0 {
			text.WriteString(f.toolValues(event.Output))
		} else {
			text.WriteString(indentTool(f.Redact(event.Text)))
		}
		if event.Kind == "tool_failed" && len(event.Output) > 0 {
			text.WriteString(indentTool("Failed: " + f.Redact(event.Text)))
		}
	}
	return text.String()
}

func (f Formatter) toolValues(body json.RawMessage) string {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		return indentTool(f.Redact(string(body)))
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var text strings.Builder
	for _, key := range keys {
		raw := fields[key]
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, raw, "", "  "); err == nil {
				value = pretty.String()
			} else {
				value = string(raw)
			}
		}
		value = f.Redact(value)
		text.WriteString(indentTool(key + ": " + value))
	}
	return text.String()
}

func indentTool(text string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(text, "\n"), "\n", "\n    ") + "\n"
}
