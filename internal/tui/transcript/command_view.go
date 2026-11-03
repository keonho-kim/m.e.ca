// Purpose: Reuse the immutable command body while refreshing elapsed-time headings.
// Pattern: Display-owner-scoped bounded projection.
// Usage: Single and team active-tool state retain one CommandView until the command settles.
// Related: internal/tui/transcript/commands.go, internal/tui/chat/team/tools.go, internal/tui/session/tool_activity.go.
package transcript

import toolapi "github.com/keonho-kim/meca/internal/agent/tool"

type CommandView struct {
	source, body string
	version      int
}

func (v *CommandView) Render(run toolapi.CommandRun, f Formatter) string {
	if f.RedactionVersion == nil {
		*v = CommandView{}
		return f.CommandActivity([]toolapi.CommandRun{run})
	}
	if run.Status != "running" {
		*v = CommandView{}
		return ""
	}
	version := f.RedactionVersion()
	body := v.body
	if v.source != run.Command || v.version != version || body == "" {
		body = indentTool(f.Redact(run.Command))
	}
	heading := f.commandHeading(run)
	text := heading + body
	*v = CommandView{}
	if len(text) <= 256*1024 {
		// The fixed body shares the actual displayed snapshot, rather than retaining another copy.
		*v = CommandView{source: run.Command, body: text[len(heading):], version: version}
	}
	return text
}
