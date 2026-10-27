// Purpose: Present compact command lists and live elapsed/last-output observations.
// Pattern: Pure command presentation.
// Usage: Native tool results and the existing terminal ticker share these projections.
// Related: internal/tui/transcript/details.go, internal/tui/session/command_activity.go.
package transcript

import (
	"encoding/json"
	"fmt"
	"strings"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func commandDuration(millis int64) string {
	seconds := max(0, millis/1000)
	return fmt.Sprintf("%02d : %02d : %02d", seconds/3600, seconds/60%60, seconds%60)
}

func (f Formatter) CommandActivity(runs []toolapi.CommandRun) string {
	var text strings.Builder
	for _, run := range runs {
		if run.Status != "running" {
			continue
		}
		text.WriteString("\n  " + f.Language.Text("Command running") + " · " + commandDuration(run.ElapsedMillis) + " · " + f.Language.Format("Last output %s ago", commandDuration(run.LastOutputMillis)) + "\n")
		text.WriteString(indentTool(brief(f.Redact(run.Command))))
	}
	return text.String()
}

func (f Formatter) commandList(body []byte) toolOutput {
	var result struct {
		Runs []toolapi.CommandRun "json:\"runs\""
	}
	if json.Unmarshal(body, &result) != nil {
		return toolOutput{}
	}
	if len(result.Runs) == 0 {
		return toolOutput{text: indentTool(f.Language.Text("No active command runs"))}
	}
	var text strings.Builder
	for _, run := range result.Runs {
		label := "Command running"
		switch run.Status {
		case "exited":
			label = "Command failed"
			if run.ExitCode != nil && *run.ExitCode == 0 {
				label = "Command succeeded"
			}
		case "interrupted":
			label = "Command interrupted"
		case "timeout":
			label = "Command timed out"
		case "execution_failed":
			label = "Command failed"
		}
		text.WriteString(indentTool(f.Redact(run.ID) + " · " + f.Language.Text(label) + " · " + commandDuration(run.ElapsedMillis)))
		text.WriteString(indentTool("  " + brief(f.Redact(run.Command))))
	}
	return toolOutput{text: text.String()}
}
