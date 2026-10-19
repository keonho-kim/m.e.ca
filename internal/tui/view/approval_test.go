// Purpose: Verify approval translation preserves literal command, path and diff values.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/view -run TestApprovalLanguage.
// Related: internal/tui/view/approval.go, internal/tui/locale/approval.go.
package view

import (
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/tui/locale"
)

func TestApprovalLanguagePreservesCommandAndPaths(t *testing.T) {
	command := `printf '%s' 'Working directory: allow-all Deletion requires approval'`
	proposal := tool.Approval{Target: command, Preview: "original record", Command: &tool.CommandApproval{
		Directory: "/project/allow-all\nTargets:", Reasons: []string{"Deletion requires approval"}, Targets: []string{"/project/Working directory: "},
	}}
	shown := ApprovalPreview(proposal, locale.Korean)
	if !strings.Contains(shown, "삭제 작업은 승인이 필요합니다") || !strings.HasSuffix(shown, "\n\n"+command) {
		t.Fatal(shown)
	}
	if !strings.Contains(shown, proposal.Command.Directory) || !strings.Contains(shown, proposal.Command.Targets[0]) || proposal.Preview != "original record" {
		t.Fatal("approval facts changed")
	}
	if got := ApprovalPreview(proposal, locale.English); !strings.HasPrefix(got, "Working directory: ") {
		t.Fatal(got)
	}
}

func TestApprovalLanguagePreservesDiff(t *testing.T) {
	preview := "--- old\n+++ new\n@@ -1 +1 @@\n-allow-all\n+Approve\n"
	if got := ApprovalPreview(tool.Approval{Diff: true, Preview: preview}, locale.Korean); got != preview {
		t.Fatal("diff changed")
	}
}
