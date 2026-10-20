// Purpose: Dispatch tool result presentation without exposing raw protocol objects.
// Pattern: Tool-specific presentation projection.
// Usage: Formatter.Tool uses the same projection for live and restored records.
// Related: internal/tui/transcript/file_results.go, internal/tui/transcript/process_result.go.
package transcript

import (
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/run"
)

func (f Formatter) output(event run.Event) string {
	body := event.Output
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return f.resultSummary(event)
	}
	text := ""
	switch event.Call.Name {
	case "execute_command":
		text = f.processOutput(fields)
	case "write_file", "edit_file", "create_directory", "move_file":
		text = f.fileOutput(event, body)
	}
	if text == "" {
		text = f.resultSummary(event)
	}
	for _, key := range []string{"Truncated", "truncated", "more"} {
		var more bool
		if json.Unmarshal(fields[key], &more) == nil && more {
			text += indentTool(f.Language.Text("[일부 결과만 표시됨]"))
			break
		}
	}
	return text
}

func (f Formatter) resultSummary(event run.Event) string {
	if event.Text != "" {
		return indentTool(f.Redact(event.Text))
	}
	return indentTool(f.Language.Text("표시할 요약 없음 · 원본 결과는 대화 기록에 보존됨"))
}
