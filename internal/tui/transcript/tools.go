// Purpose: Format literal tool details with redaction and explicit source-change annotations.
// Pattern: Presentation projection.
// Usage: ToolBlock formats summary or requested detail; Event projects completed tool records.
// Related: internal/agent/record/events.go, internal/tui/transcript/events.go.
package transcript

import (
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (f Formatter) Tool(event runrecord.Event) string {
	return f.toolBlock(event).Text
}

// ToolBlock parses display arguments once for the selected summary/detail representation.
func (f Formatter) ToolBlock(event runrecord.Event, expanded bool) viewport.Block {
	if event.Call == nil {
		return viewport.Block{}
	}
	args, err := toolapi.DisplayArgs(event.Call.Arguments)
	summary := f.toolSummaryWithArguments(event, args, err)
	if !expanded {
		return viewport.Block{Text: summary, Summary: summary}
	}
	arguments := indentTool(f.Language.Text("인자 형식 확인 필요"))
	if err == nil {
		arguments = f.displayArguments(args)
	}
	block := f.toolBlockWithArguments(event, arguments)
	if block.Text != "" {
		block.Summary, block.Details = summary, block.Text
	}
	return block
}

func (f Formatter) toolBlock(event runrecord.Event) viewport.Block {
	if event.Call == nil {
		return viewport.Block{}
	}
	return f.toolBlockWithArguments(event, f.arguments(*event.Call))
}

func (f Formatter) toolBlockWithArguments(event runrecord.Event, arguments string) viewport.Block {
	var text strings.Builder
	var changes []viewport.ChangeRange
	label := f.Language.Text(event.Call.Name)
	if event.Kind == "tool_started" || event.Kind == "tool_waiting" {
		status := "Running"
		if event.Kind == "tool_waiting" {
			status = "실행 대기"
		}
		return viewport.Block{Text: "\n  " + label + " · " + f.Language.Text(status) + "…\n" + arguments}
	}
	text.WriteString("\n  " + label + "\n\n")
	text.WriteString(arguments)
	if event.Call.Name != "execute_command" && event.Operation != nil && event.Operation.Process != nil && event.Operation.Process.Command != "" {
		text.WriteString(indentTool(f.Redact(event.Operation.Process.Command)))
	}
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
