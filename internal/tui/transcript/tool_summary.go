// Purpose: Summarize collapsed targets and localized outcomes without raw command metadata.
// Pattern: Pure presentation projection.
// Usage: Formatter.Event supplies summary and full detail blocks for every completed tool.
// Related: internal/tui/transcript/events.go, internal/tui/transcript/tools.go.
package transcript

import (
	"encoding/json"
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/rivo/uniseg"
)

func brief(text string) string {
	text = strings.NewReplacer("\r", "", "\n", " ↵ ", "\t", " ").Replace(text)
	graphemes := uniseg.NewGraphemes(text)
	for count := 0; graphemes.Next(); count++ {
		if count == 120 {
			start, _ := graphemes.Positions()
			return text[:start] + "…"
		}
	}
	return text
}

func (f Formatter) toolSummary(event runrecord.Event) string {
	status := map[string]string{"tool_succeeded": "완료", "tool_failed": "실패", "tool_interrupted": "중단", "denied": "거부"}[event.Kind]
	commandStatus, commandSummary := "", ""
	if event.Call.Name == "execute_command" {
		var fields map[string]json.RawMessage
		if json.Unmarshal(event.Output, &fields) == nil {
			commandStatus = processStatus(event.Kind, fields)
			if commandStatus != "" {
				status = commandStatus
				commandSummary = f.processSummary(event, fields)
			}
		}
	}
	text := "\n  " + f.Language.Text(event.Call.Name) + " · " + f.Language.Text(status) + " · " + f.Language.Text("Collapsed") + "\n"
	var args struct {
		Path, Command string
		Paths         []string
	}
	if json.Unmarshal(event.Call.Arguments, &args) == nil {
		target := args.Path
		if args.Command != "" {
			target = args.Command
		}
		if len(args.Paths) > 0 {
			target = args.Paths[0]
			if len(args.Paths) > 1 {
				target += f.Language.Format(" 외 %d개", len(args.Paths)-1)
			}
		}
		if target != "" {
			text += indentTool(brief(f.Redact(target)))
		}
	}
	var output struct {
		Changed   *bool
		Created   []string
		SizeBytes *int `json:"size_bytes"`
	}
	result := event.Text
	if commandStatus != "" {
		result = commandSummary
	} else if json.Unmarshal(event.Output, &output) == nil {
		switch {
		case event.Kind == "tool_succeeded" && output.Changed != nil && !*output.Changed:
			result = f.Language.Text("변경 없음")
		case event.Kind == "tool_succeeded" && len(output.Created) > 0:
			result = f.Language.Format("생성한 폴더 %d개", len(output.Created))
		case event.Kind == "tool_succeeded" && output.SizeBytes != nil:
			result = f.Language.Format("%d bytes", *output.SizeBytes)
		}
	}
	if result != "" {
		text += indentTool(brief(f.Redact(result)))
	}
	return text + "    [Ctrl+O: " + f.Language.Text("Expand") + "]\n"
}
