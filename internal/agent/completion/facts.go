// Purpose: Project execution receipts into a final report without replaying tool transcripts.
// Pattern: Pure projection.
// Usage: Runtime finalization and interrupted-work recovery derive Facts from recorded receipts.
// Related: internal/agent/completion/state.go, internal/agent/tool/facts.go.
package completion

import (
	"github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s State) Facts() tool.Facts {
	files := map[string]tool.FileFact{}
	result := tool.Facts{}
	for _, r := range s.Receipts {
		for path, version := range r.Versions {
			prior := files[path]
			files[path] = tool.FileFact{Path: path, Version: version, Changed: prior.Changed || r.Changed, Stale: r.Stale}
		}
		if r.Kind == "command" {
			command, c := tool.ReportExcerpt(r.Command)
			output, o := tool.ReportExcerpt(r.Output)
			result.Commands = append(result.Commands, tool.CommandFact{CallID: r.ID, Command: command, Status: r.ProcessStatus, ExitCode: r.ExitCode, Output: output, Excerpt: c || o, ObservedRun: r.ObservedRun, Revision: r.Revision, BeforeLaterChanges: r.BeforeLaterChanges})
		}
	}
	var own tool.Facts
	for _, f := range files {
		own.Files = append(own.Files, f)
	}
	result.Merge(own)
	if len(s.CurrentFiles) > 0 {
		result.Files = append([]tool.FileFact(nil), s.CurrentFiles...)
	}
	return result
}
