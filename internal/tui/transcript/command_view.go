// Purpose: Reuse the selected folded or full command body while refreshing elapsed-time headings.
// Pattern: Display-owner-scoped bounded projection.
// Usage: Single-mode live activity retains one CommandView until the command settles.
// Related: internal/tui/transcript/commands.go, internal/tui/session/tool_activity.go.
package transcript

import toolapi "github.com/keonho-kim/meca/internal/agent/tool"

type CommandView struct {
	source, body string
	version      int
	collapsed    bool
}

func (v *CommandView) Render(run toolapi.CommandRun, f Formatter, collapsed bool) string {
	version := -1
	if f.RedactionVersion != nil {
		version = f.RedactionVersion()
	}
	active := toolapi.CommandActive(run.Status)
	body := v.body
	if !active || version < 0 || v.source != run.Command || v.version != version || v.collapsed != collapsed || body == "" {
		command := f.Redact(run.Command)
		if collapsed {
			command = brief(command)
		}
		body = indentTool(command)
	}
	heading := f.commandHeading(run)
	text := heading + body
	*v = CommandView{}
	if active && version >= 0 && len(text) <= 256*1024 {
		// The fixed body shares the actual displayed snapshot, rather than retaining another copy.
		*v = CommandView{source: run.Command, body: text[len(heading):], version: version, collapsed: collapsed}
	}
	return text
}
