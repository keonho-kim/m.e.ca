// Purpose: Localize approval context while preserving command text, target paths and frozen diffs.
// Pattern: Pure presentation projection.
// Usage: The session supplies an immutable prepared approval to ApprovalPreview.
// Related: internal/agent/tool/execution.go, internal/tui/session/render_menu.go.
package view

import (
	"strings"

	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func ApprovalPreview(proposal tool.Approval, lang locale.Language) string {
	if command := proposal.Command; command != nil {
		lines := []string{lang.Text("Working directory: ") + command.Directory}
		for _, finding := range command.Findings {
			name := finding.Command
			if name == "" {
				name = lang.Text("Executable decided at runtime")
			}
			if name == "redirection" {
				name = lang.Text("Output redirection")
			}
			lines = append(lines, lang.Text("Command: ")+name)
			for _, reason := range finding.Reasons {
				lines = append(lines, "  "+lang.Text(reason))
			}
			if len(finding.Targets) > 0 {
				lines = append(lines, "  "+lang.Text("Targets:"))
				for _, path := range finding.Targets {
					lines = append(lines, "  "+path)
				}
			}
		}
		return strings.Join(lines, "\n") + "\n\n" + proposal.Target
	}
	if proposal.Diff {
		return proposal.Preview
	}
	// File previews have a fixed authored prefix followed by literal paths.
	for _, prefix := range []string{"Move to ", "Create directories (including missing parents):\n"} {
		if tail, ok := strings.CutPrefix(proposal.Preview, prefix); ok {
			return lang.Text(prefix) + tail
		}
	}
	if proposal.Preview == "All requested directories already exist." {
		return lang.Text(proposal.Preview)
	}
	return proposal.Preview
}
