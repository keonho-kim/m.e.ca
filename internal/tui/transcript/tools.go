// Purpose: Format literal tool details with redaction and explicit source-change annotations.
// Pattern: Presentation projection.
// Usage: Live status uses Formatter.Tool; completed and archived events retain toolBlock annotations.
// Related: internal/agent/record/events.go, internal/tui/transcript/events.go.
package transcript

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
	"strings"
)

func (f Formatter) Tool(event runrecord.Event) string {
	return f.toolBlock(event).Text
}

func (f Formatter) toolBlock(event runrecord.Event) viewport.Block {
	if event.Call == nil {
		return viewport.Block{}
	}
	var text strings.Builder
	var changes []viewport.ChangeRange
	label := f.Language.Text(event.Call.Name)
	if event.Kind == "tool_started" {
		return viewport.Block{Text: "\n  " + label + " · " + f.Language.Text("Running") + "…\n" + f.arguments(*event.Call)}
	}
	text.WriteString("\n  " + label + "\n\n")
	text.WriteString(f.arguments(*event.Call))
	text.WriteString("\n    " + f.Language.Text("Output") + "\n")
	rendered := f.output(event)
	if len(event.Output) > 0 {
		for _, change := range rendered.changes {
			change.Start += text.Len()
			change.End += text.Len()
			changes = append(changes, change)
		}
		text.WriteString(rendered.text)
	} else {
		text.WriteString(indentTool(f.Redact(event.Text)))
	}
	if event.Kind == "tool_failed" && len(event.Output) > 0 && !rendered.failureHandled && rendered.text != f.resultSummary(event) {
		text.WriteString(indentTool(f.Language.Text("Failed: ") + f.Redact(event.Text)))
	}
	block := viewport.Block{Text: text.String()}
	if len(changes) > 0 {
		block.Changes = &viewport.Changes{Source: block.Text, Ranges: changes}
	}
	return block
}

func indentTool(text string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(text, "\n"), "\n", "\n    ") + "\n"
}
