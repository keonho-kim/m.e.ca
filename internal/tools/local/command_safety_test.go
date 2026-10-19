// Purpose: Verify command safety blocks protected targets and cannot be bypassed by automatic approval.
// Pattern: Tool policy regression tests.
// Usage: go test ./internal/tools/local -run TestCommandSafety.
// Related: internal/tools/local/prepare.go, internal/tools/local/operation.go.
package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/v-code/internal/policy"
)

func TestCommandSafetyBlocksProtectedTargets(t *testing.T) {
	for _, command := range []string{
		`rm -rf /etc`,
		`env command rm -rf "$HOME/bin"`,
		`TARGET=/etc; rm -rf "$TARGET"`,
		`cd /etc && rm -rf .`,
		`echo "$(rm -rf /etc)"`,
		`sh -c 'rm -rf /etc'`,
		`eval 'rm -rf /etc'`,
		`find /etc -delete`,
		`sudo true`,
	} {
		t.Run(command, func(t *testing.T) {
			e, _ := testExecutor(t)
			t.Setenv("HOME", e.home)
			_, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.AllowAll}, call("execute_command", fmt.Sprintf(`{"command":%q}`, command)))
			if err == nil {
				t.Fatal("protected command was prepared")
			}
		})
	}
}

func TestCommandSafetyRequiresApprovalUnderAllowAll(t *testing.T) {
	e, project := testExecutor(t)
	path := filepath.Join(project, "remove-me")
	if err := os.WriteFile(path, []byte("preserve until approved"), 0600); err != nil {
		t.Fatal(err)
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.AllowAll}, call("execute_command", `{"command":"rm -f remove-me"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !op.Approval().Required {
		t.Fatal("destructive command bypassed approval")
	}
	if _, err := op.Execute(context.Background(), false); err == nil {
		t.Fatal("unapproved operation executed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("unapproved deletion changed the file")
	}
	result, err := op.Execute(context.Background(), true)
	if err != nil || result.Failed {
		t.Fatalf("approved deletion failed: %+v %v", result, err)
	}
}
