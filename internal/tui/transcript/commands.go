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
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

// CommandBlock reconstructs an evicted pending card without parsing native arguments again.
func (f Formatter) CommandBlock(run toolapi.CommandRun) viewport.Block {
	command := f.Redact(run.Command)
	heading := f.commandHeading(run)
	text := heading + indentTool(command)
	return viewport.Block{Text: text, Summary: f.commandSummary(run, command), Details: text}
}

// CommandSummary projects current status without loading or replacing captured output.
func (f Formatter) CommandSummary(run toolapi.CommandRun) string {
	return f.commandSummary(run, f.Redact(run.Command))
}

func (f Formatter) commandSummary(run toolapi.CommandRun, command string) string {
	return "\n  " + f.Language.Text("execute_command") + " · " + f.Language.Text(commandLabel(run)) + " · " + f.Language.Text("Collapsed") + "\n" + indentTool(brief(command))
}

func commandDuration(millis int64) string {
	seconds := max(0, millis/1000)
	return fmt.Sprintf("%02d : %02d : %02d", seconds/3600, seconds/60%60, seconds%60)
}

func (f Formatter) CommandActivity(runs []toolapi.CommandRun) string {
	var text strings.Builder
	for _, run := range runs {
		if !toolapi.CommandActive(run.Status) {
			continue
		}
		text.WriteString(f.commandHeading(run))
		text.WriteString(indentTool(f.Redact(run.Command)))
	}
	return text.String()
}

func (f Formatter) commandHeading(run toolapi.CommandRun) string {
	label := commandLabel(run)
	if !toolapi.CommandActive(run.Status) {
		return "\n  " + f.Language.Text(label) + " · " + f.Language.Format("Elapsed %s", commandDuration(run.ElapsedMillis)) + "\n"
	}
	last := f.Language.Text("No output yet")
	if run.LastOutputMillis != nil {
		last = f.Language.Format("Last output %s ago", commandDuration(*run.LastOutputMillis))
	}
	return "\n  " + f.Language.Text(label) + " · " + f.Language.Format("Elapsed %s", commandDuration(run.ElapsedMillis)) + " · " + last + "\n"
}

func commandLabel(run toolapi.CommandRun) string {
	switch run.Status {
	case "running":
		return "Command running"
	case "waiting":
		return "Command waiting"
	case "finalizing":
		return "Finalizing command output"
	case "interrupted":
		return "Command interrupted"
	case "timeout":
		return "Command timed out"
	case "exited":
		if run.ExitCode != nil && *run.ExitCode == 0 {
			return "Command succeeded"
		}
	}
	return "Command failed"
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
		label := commandLabel(run)
		text.WriteString(indentTool(f.Redact(run.ID) + " · " + f.Language.Text(label) + " · " + commandDuration(run.ElapsedMillis)))
		text.WriteString(indentTool("  " + brief(f.Redact(run.Command))))
	}
	return toolOutput{text: text.String()}
}
