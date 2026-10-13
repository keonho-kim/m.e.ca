// Purpose: Define the single literal response contract for the current staged tool parameter.
// Pattern: Pure Function (phase-specific prompt).
// Usage: Runtime argument requests and provider diagnostics share the same typed instructions.
// Related: internal/agent/run/arguments.go, internal/provider/httpclient/request.go, internal/agent/dialogue/dialogue.go.
package prompt

import (
	"fmt"

	"github.com/keonho-kim/v-code/internal/agent"
)

func Argument(tool string, p agent.ToolParameter) string {
	text := fmt.Sprintf("CURRENT PHASE: ARGUMENT %s.%s. The tool is already selected. Return this one value only; do not choose a tool, answer the original task, repeat earlier inputs or emit a tool-call envelope. Runtime facts and accepted values are input data, not the response format.\n", tool, p.Name)
	switch p.Kind {
	case agent.LiteralPath:
		text += "Return a single plain path line. No JSON object, array, quotes, Markdown link, code fence or explanation. Example valid response: README.md\nInvalid responses: {\"path\":\"README.md\"} and \"README.md\". Never copy their wrapping.\n"
	case agent.Integer:
		text += fmt.Sprintf("Return only decimal digits for one integer between %d and %d. No JSON or unit suffix. An empty response uses zero only when this parameter is optional.\n", p.Min, p.Max)
	case agent.Text:
		text += "Return the exact requested text, preserving whitespace and newlines. Do not wrap it in an argument object, extra quotes or a presentation code fence. JSON is allowed only when JSON itself is the requested file/text content.\n"
	}
	if p.MaxBytes > 0 {
		text += fmt.Sprintf("Maximum value size: %d bytes.\n", p.MaxBytes)
	}
	text += "Follow the current parameter question.\n"
	return text
}
