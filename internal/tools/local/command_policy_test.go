// Purpose: Verify shell calls accept only command text and retain Scope and Permission approval guards.
// Pattern: Tool boundary behavior tests.
// Usage: go test ./internal/tools/local -run TestCommandContract.
// Related: internal/tools/local/catalog.go, internal/tools/local/prepare.go, internal/tools/local/operation.go.
package local

import (
	"context"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"testing"
)

func TestCommandContractRetainsPolicyWithoutTimeArgument(t *testing.T) {
	executor, root := testExecutor(t)
	for _, definition := range executor.Definitions() {
		if definition.Name == "execute_command" && (len(definition.Parameters) != 1 || definition.Parameters[0].Name != "command") {
			t.Fatalf("unexpected command fields: %+v", definition.Parameters)
		}
	}
	if _, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.AllowAll}, call("execute_command", `{"command":"true","purpose":"check"}`)); err == nil {
		t.Fatal("removed purpose field accepted")
	}
	ctx := context.Background()
	command := call("execute_command", `{"command":"printf approved > marker"}`)
	settings := policy.Settings{Permission: policy.Approve, Scope: policy.System}
	if _, err := executor.Prepare(ctx, settings, call("execute_command", `{"command":"true","timeout_seconds":1}`)); err == nil {
		t.Fatal("obsolete time field accepted")
	}
	op, err := executor.Prepare(ctx, settings, command)
	if err != nil {
		t.Fatal(err)
	}
	if !op.Approval().Required || op.Approval().Preview != "printf approved > marker" {
		t.Fatal("command approval missing")
	}
	if _, err := op.Execute(ctx, false); err == nil {
		t.Fatal("unapproved command executed")
	}
	if _, err := os.Stat(filepath.Join(root, "marker")); !os.IsNotExist(err) {
		t.Fatalf("rejected command changed files: %v", err)
	}
	if result, err := op.Execute(ctx, true); err != nil || result.Failed {
		t.Fatalf("approved command failed: %+v %v", result, err)
	}
	content, err := os.ReadFile(filepath.Join(root, "marker"))
	if err != nil || string(content) != "approved" {
		t.Fatalf("execution output=%q %v", content, err)
	}
}
