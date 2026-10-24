// Purpose: Verify ordinary shell calls execute automatically under the host Scope contract.
// Pattern: Tool boundary behavior tests.
// Usage: go test ./internal/tools/local -run TestCommandContract.
// Related: internal/tools/local/catalog.go, internal/tools/local/prepare.go, internal/tools/local/operation.go.
package local

import (
	"context"
	"github.com/keonho-kim/meca/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandContractUsesAutomaticApproval(t *testing.T) {
	executor, root := testExecutor(t)
	for _, definition := range executor.Definitions() {
		if definition.Name == "execute_command" && (len(definition.Parameters) != 1 || definition.Parameters[0].Name != "command") {
			t.Fatalf("unexpected command fields: %+v", definition.Parameters)
		}
	}
	ctx := context.Background()
	command := call("execute_command", `{"command":"git init"}`)
	if _, err := executor.Prepare(ctx, policy.Settings{Scope: policy.Directory}, command); err == nil {
		t.Fatal("host shell accepted a directory-confinement contract")
	}
	settings := policy.Settings{Scope: policy.System}
	op, err := executor.Prepare(ctx, settings, command)
	if err != nil {
		t.Fatal(err)
	}
	if op.Approval().Required || !strings.Contains(op.Approval().Preview, "git init") {
		t.Fatal("ordinary command requires approval or lost its preview")
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); !os.IsNotExist(err) {
		t.Fatalf("rejected command changed files: %v", err)
	}
	if result, err := op.Execute(ctx, false); err != nil || result.Failed {
		t.Fatalf("approved command failed: %+v %v", result, err)
	}
	if info, err := os.Stat(filepath.Join(root, ".git")); err != nil || !info.IsDir() {
		t.Fatalf("approved package operation did not run: %v", err)
	}
}
