// Purpose: Verify real command outcomes reach model context and observation reports without losing failure facts.
// Pattern: Runtime and process integration test.
// Usage: go test ./internal/agent/run -run TestCommandFailureReachesModel.
// Related: internal/agent/run/action.go, internal/tools/local/command.go, internal/agent/tool/failure.go.
package run

import (
	"context"
	"encoding/json"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/tools/local"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandFailureReachesModel(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	tools, err := local.NewExecutor(project, home, filepath.Join(home, ".v"), func(s string) string { return s })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tools.Close(); err != nil {
			t.Error(err)
		}
	})
	g := &generator{replies: []string{nativeCall("execute_command", map[string]any{"command": "printf progress; printf 'missing dependency' >&2; exit 7"}), "Need to resolve the missing dependency", "Cannot verify until the dependency is available"}}
	j := &journal{}
	r := fixture(g, &executor{}, j)
	r.Tools = tools
	r.Settings.Scope = policy.System
	approvals := 0
	result, err := r.Run(context.Background(), "run check", func(context.Context, toolapi.Approval) (bool, error) { approvals++; return true, nil }, emit)
	if err != nil || result.Status != Answered || approvals != 1 || len(g.requests) != 2 {
		t.Fatalf("result=%+v err=%v approvals=%d requests=%d", result, err, approvals, len(g.requests))
	}
	found := false
	for _, message := range j.messages {
		if message.Role != "tool" {
			continue
		}
		var body struct {
			Failure *toolapi.Failure
			Output  struct {
				Stdout, Stderr, Status string
				ExitCode               int
			}
		}
		if err := json.Unmarshal([]byte(message.Content), &body); err != nil {
			t.Fatal(err)
		}
		if body.Failure == nil || body.Failure.Code != toolapi.ProcessFailed || body.Failure.Next == "" || body.Output.ExitCode != 7 || body.Output.Stderr != "missing dependency" || body.Output.Stdout != "progress" {
			t.Fatalf("failure information lost: %+v", body)
		}
		found = true
	}
	var contextText strings.Builder
	for _, message := range g.requests[1].Messages {
		contextText.WriteString(message.Content)
	}
	if !found || !strings.Contains(contextText.String(), "missing dependency") || !strings.Contains(contextText.String(), toolapi.ProcessFailed) {
		t.Fatal("failure omitted from next model context")
	}
}

func TestNoMatchSearchFinishesWithoutCorrectionOrRetry(t *testing.T) {
	r, _, root, _, _ := realBatchRunner(t, nil)
	if err := os.WriteFile(filepath.Join(root, "app.txt"), []byte("done\n"), 0600); err != nil {
		t.Fatal(err)
	}
	g := &generator{replies: []string{nativeCall("execute_command", map[string]any{"command": "grep TODO app.txt"}), "No TODO matches were found; grep exited with code 1."}}
	r.Generator = g
	result, err := r.Run(context.Background(), "Search app.txt for TODO and report whether it occurs", approve, emit)
	if err != nil || result.Status != Answered || len(g.requests) != 2 {
		t.Fatalf("unnecessary recovery: %+v %v calls=%d", result, err, len(g.requests))
	}
	if len(result.Facts.Commands) != 1 || result.Facts.Commands[0].ExitCode == nil || *result.Facts.Commands[0].ExitCode != 1 || !result.HadToolFailure {
		t.Fatalf("observation overwrote command failure facts: %+v", result)
	}
}
