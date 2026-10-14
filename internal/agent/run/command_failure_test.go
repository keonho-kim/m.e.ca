// Purpose: Verify a real shell failure reaches the next model request with its diagnostics.
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
	g := &generator{replies: []string{`{"tool":"execute_command"}`, "printf progress; printf 'missing dependency' >&2; exit 7", "WAIT\nNeed to resolve the missing dependency"}}
	j := &journal{}
	r := fixture(g, &executor{}, j)
	r.Tools = tools
	r.Settings.Scope = policy.System
	approvals := 0
	result, err := r.Run(context.Background(), "run check", false, func(context.Context, toolapi.Approval) (bool, error) { approvals++; return true, nil }, emit)
	if err != nil || result.Status != NeedsInput || approvals != 1 || len(g.requests) != 3 {
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
	for _, message := range g.requests[2].Messages {
		contextText.WriteString(message.Content)
	}
	if !found || !strings.Contains(contextText.String(), "missing dependency") || !strings.Contains(contextText.String(), toolapi.ProcessFailed) {
		t.Fatal("failure omitted from next model context")
	}
}

func TestProjectShellScopeFailureReachesModel(t *testing.T) {
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
	target := filepath.Join(t.TempDir(), "outside")
	g := &generator{replies: []string{`{"tool":"execute_command"}`, "printf outside > '" + target + "'", "WAIT\nThe outside path is unavailable under current Scope"}}
	j := &journal{}
	r := fixture(g, &executor{}, j)
	r.Tools = tools
	r.Settings.Permission = policy.AllowAll
	result, err := r.Run(context.Background(), "inspect execution limits", false, approve, emit)
	if err != nil || result.Status != NeedsInput || !result.HadToolFailure {
		t.Fatalf("result=%+v %v", result, err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("scope escape: %v", err)
	}
	var body strings.Builder
	for _, message := range g.requests[2].Messages {
		body.WriteString(message.Content)
	}
	if !strings.Contains(body.String(), "directory") || !strings.Contains(body.String(), "Failure") || !strings.Contains(body.String(), "Output") {
		t.Fatal("policy or execution error missing from model context")
	}
}
