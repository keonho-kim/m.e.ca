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
		Directory: "/project/allow-all\nTargets:", Findings: []tool.CommandFinding{{Command: "rm", Reasons: []string{"Deletion requires approval"}, Targets: []string{"/project/Working directory: "}}},
	}}
	shown := ApprovalPreview(proposal, locale.Korean)
	if !strings.Contains(shown, "삭제 작업은 승인이 필요합니다") || !strings.HasSuffix(shown, "\n\n"+command) {
		t.Fatal(shown)
	}
	if !strings.Contains(shown, proposal.Command.Directory) || !strings.Contains(shown, proposal.Command.Findings[0].Targets[0]) || proposal.Preview != "original record" {
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

func TestApprovalLanguageGroupsRiskySteps(t *testing.T) {
	proposal := tool.Approval{Target: "printf x > file && rm old && sed edit", Command: &tool.CommandApproval{Directory: "/project", Findings: []tool.CommandFinding{
		{Command: "rm", Reasons: []string{"Deletion requires approval"}, Targets: []string{"/project/old"}},
		{Command: "sed", Reasons: []string{"Mutation outside the project requires approval"}, Targets: []string{"/outside/file"}},
		{Reasons: []string{"Unresolved command name requires approval"}},
	}}}
	shown := ApprovalPreview(proposal, locale.Korean)
	for _, text := range []string{"검토할 명령: rm\n  삭제 작업은 승인이 필요합니다", "검토할 명령: sed\n  프로젝트 밖 변경은 승인이 필요합니다", "검토할 명령: 실행 중 결정되는 명령", "/project/old", "/outside/file"} {
		if !strings.Contains(shown, text) {
			t.Fatal(shown)
		}
	}
	if !strings.HasSuffix(shown, "\n\n"+proposal.Target) {
		t.Fatal("review did not preserve the exact full command")
	}
}
