// Purpose: Verify directory-aware approval matches real shell writes and preserves denial before execution.
// Pattern: Temporary-filesystem command integration tests.
// Usage: go test ./internal/tools/local -run TestCommandDirectory.
// Related: internal/tools/local/command_safety.go, internal/tools/local/command.go, internal/policy/command/directories.go.
package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func TestCommandDirectoryBranchesReachTheActualDestination(t *testing.T) {
	for _, test := range []struct{ command, destination string }{
		{`cd child; printf result > result.txt`, "child/result.txt"},
		{`cd missing; printf result > result.txt`, "result.txt"},
		{`cd child && printf 'done\n' | tail -n 1; printf result > result.txt`, "child/result.txt"},
		{`sh -c 'cd child; printf result > result.txt'`, "child/result.txt"},
	} {
		t.Run(test.command, func(t *testing.T) {
			executor, project := testExecutor(t)
			if err := os.Mkdir(filepath.Join(project, "child"), 0700); err != nil {
				t.Fatal(err)
			}
			op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, test.command)))
			if err != nil {
				t.Fatal(err)
			}
			if op.Approval().Required {
				t.Fatalf("internal directory branches required review: %+v", op.Approval())
			}
			result, err := op.Execute(context.Background(), false)
			if err != nil || result.Failed {
				t.Fatalf("command did not complete: %+v %v", result, err)
			}
			body, err := os.ReadFile(filepath.Join(project, test.destination))
			if err != nil || string(body) != "result" {
				t.Fatalf("wrong shell destination: %q %v", body, err)
			}
			if test.destination != "result.txt" {
				if _, err := os.Stat(filepath.Join(project, "result.txt")); !os.IsNotExist(err) {
					t.Fatalf("shell was reset to the project root inside a call: %v", err)
				}
			}
		})
	}
}

func TestCommandDirectoryExternalBranchNeedsApproval(t *testing.T) {
	executor, project := testExecutor(t)
	external := t.TempDir()
	text := fmt.Sprintf("cd %q; printf changed > result.txt", external)
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, text)))
	if err != nil {
		t.Fatal(err)
	}
	if !op.Approval().Required {
		t.Fatal("possible external working directory was automatically approved")
	}
	if _, err := op.Execute(context.Background(), false); err == nil {
		t.Fatal("unapproved command ran")
	}
	for _, directory := range []string{project, external} {
		if _, err := os.Stat(filepath.Join(directory, "result.txt")); !os.IsNotExist(err) {
			t.Fatalf("denial allowed a write in %s: %v", directory, err)
		}
	}
}

func TestCommandDirectoryProtectedBranchIsBlocked(t *testing.T) {
	executor, _ := testExecutor(t)
	_, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"cd /etc; printf changed > v-test-result"}`))
	var failure *toolapi.Failure
	if !errors.As(err, &failure) || failure.Code != toolapi.CommandBlocked {
		t.Fatalf("known system directory branch was not blocked: %v", err)
	}
}
