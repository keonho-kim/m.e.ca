// Purpose: Verify shell calls accept only command text and retain Scope and Permission approval guards.
// Pattern: Tool boundary behavior tests.
// Usage: go test ./internal/tools/local -run TestCommandContract.
// Related: internal/tools/local/catalog.go, internal/tools/local/prepare.go, internal/tools/local/operation.go.
package local

import (
	"context"
	"errors"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandContractRequiresPermission(t *testing.T) {
	executor, root := testExecutor(t)
	for _, definition := range executor.Definitions() {
		if definition.Name == "execute_command" && (len(definition.Parameters) != 1 || definition.Parameters[0].Name != "command") {
			t.Fatalf("unexpected command fields: %+v", definition.Parameters)
		}
	}
	ctx := context.Background()
	command := call("execute_command", `{"command":"git init"}`)
	settings := policy.Settings{Permission: policy.Approve, Scope: policy.System}
	op, err := executor.Prepare(ctx, settings, command)
	if err != nil {
		t.Fatal(err)
	}
	if !op.Approval().Required || !strings.Contains(op.Approval().Preview, "git init") {
		t.Fatal("command approval missing")
	}
	if _, err := op.Execute(ctx, false); err == nil {
		t.Fatal("unapproved command executed")
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); !os.IsNotExist(err) {
		t.Fatalf("rejected command changed files: %v", err)
	}
	if result, err := op.Execute(ctx, true); err != nil || result.Failed {
		t.Fatalf("approved command failed: %+v %v", result, err)
	}
	if info, err := os.Stat(filepath.Join(root, ".git")); err != nil || !info.IsDir() {
		t.Fatalf("approved package operation did not run: %v", err)
	}
}

func TestMixedCommandRejectsEverythingWithSplitAdvice(t *testing.T) {
	executor, root := testExecutor(t)
	_, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.AllowAll}, call("execute_command", `{"command":"printf started > marker && cat pyproject.toml"}`))
	var failure *toolapi.Failure
	if !errors.As(err, &failure) || failure.Code != "use_dedicated_tool" {
		t.Fatalf("missing routing failure: %v", err)
	}
	for _, expected := range []string{"Nothing from the rejected shell command ran", "separate", "write_file"} {
		if !strings.Contains(failure.Next, expected) {
			t.Fatalf("missing %q in recovery advice: %s", expected, failure.Next)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); !os.IsNotExist(err) {
		t.Fatalf("rejected command partially executed: %v", err)
	}
}
