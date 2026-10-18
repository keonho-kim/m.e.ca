// Purpose: Format tool parameters and actual results with indentation and literal-value redaction.
// Pattern: Presentation projection.
// Usage: Live and archived tool events use Formatter.Tool.
// Related: internal/agent/run/events.go, internal/tui/transcript/events.go.
package transcript

import (
	"github.com/keonho-kim/v-code/internal/agent/run"
	"strings"
)

func (f Formatter) Tool(event run.Event) string {
	if event.Call == nil {
		return ""
	}
	var text strings.Builder
	label := event.Call.Name
	text.WriteString("\n  " + label + "\n\n")
	text.WriteString(f.arguments(*event.Call))
	if event.Kind == "tool_started" {
		text.WriteString("\n    Running…\n")
	} else {
		text.WriteString("\n    Output\n")
		rendered := f.output(event)
		if len(event.Output) > 0 {
			text.WriteString(rendered)
		} else {
			text.WriteString(indentTool(f.Redact(event.Text)))
		}
		if event.Kind == "tool_failed" && len(event.Output) > 0 && rendered != f.resultSummary(event) {
			text.WriteString(indentTool("Failed: " + f.Redact(event.Text)))
		}
	}
	return text.String()
}

func indentTool(text string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(text, "\n"), "\n", "\n    ") + "\n"
}
