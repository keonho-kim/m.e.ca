// Purpose: Verify automatic project-child deletion preserves external approval and protected roots.
// Pattern: Temporary-filesystem policy integration tests.
// Usage: go test ./internal/tools/local -run TestDeletionPolicy.
// Related: internal/policy/safety.go, internal/tools/local/command_safety.go, internal/tools/local/operation.go.
package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestDeletionPolicyProjectChildrenRunAutomatically(t *testing.T) {
	e, project := testExecutor(t)
	for _, directory := range []string{"nested/deep", "empty"} {
		if err := os.MkdirAll(filepath.Join(project, directory), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"sample.py", "unlink-me", "nested/deep/file"} {
		if err := os.WriteFile(filepath.Join(project, name), []byte("temporary"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"rm sample.py && unlink unlink-me && rm -rf nested && rmdir empty"}`))
	if err != nil {
		t.Fatal(err)
	}
	if op.Approval().Required {
		t.Fatalf("project children required approval: %+v", op.Approval())
	}
	result, err := op.Execute(context.Background(), false)
	if err != nil || result.Failed {
		t.Fatalf("automatic deletion failed: %+v %v", result, err)
	}
	for _, name := range []string{"sample.py", "unlink-me", "nested", "empty"} {
		if _, err := os.Lstat(filepath.Join(project, name)); !os.IsNotExist(err) {
			t.Fatalf("%s remains: %v", name, err)
		}
	}
	if info, err := os.Stat(project); err != nil || !info.IsDir() {
		t.Fatalf("project root changed: %v", err)
	}
}

func TestDeletionPolicyBoundaryDecisions(t *testing.T) {
	e, project := testExecutor(t)
	for _, test := range []struct {
		command           string
		approval, blocked bool
	}{
		{fmt.Sprintf("rm %q", filepath.Join(project, "sample.py")), false, false},
		{fmt.Sprintf("rm %q", project+"-other/file"), true, false},
		{`rm "$(cat target)"`, true, false},
		{`rm *.tmp`, true, false},
		{`rm -rf .`, false, true},
		{`rm -rf ..`, false, true},
		{fmt.Sprintf("rm -rf %q", project), false, true},
		{`rm -rf /etc`, false, true},
	} {
		t.Run(test.command, func(t *testing.T) {
			op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, test.command)))
			if (err != nil) != test.blocked {
				t.Fatalf("blocked=%v: %v", test.blocked, err)
			}
			if err == nil && op.Approval().Required != test.approval {
				t.Fatalf("approval=%v: %+v", test.approval, op.Approval())
			}
		})
	}
}

func TestAllMightyPreservesProtectedTargetChecks(t *testing.T) {
	e, _ := testExecutor(t)
	settings := policy.Settings{Scope: policy.System, Permission: policy.PermissionAllMighty}
	for _, command := range []string{"rm -rf .", "rm -rf /etc", "sudo true"} {
		if _, err := e.Prepare(context.Background(), settings, call("execute_command", fmt.Sprintf(`{"command":%q}`, command))); err == nil {
			t.Fatalf("automatic approval bypassed a protected operation: %s", command)
		}
	}
}

func TestDeletionPolicyChangedAliasRequiresPreparation(t *testing.T) {
	e, project := testExecutor(t)
	inside, outside := filepath.Join(project, "inside"), t.TempDir()
	if err := os.Mkdir(inside, 0700); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{inside, outside} {
		if err := os.WriteFile(filepath.Join(directory, "file"), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(project, "alias")
	if err := os.Symlink(inside, alias); err != nil {
		t.Fatal(err)
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"rm alias/file"}`))
	if err != nil || op.Approval().Required {
		t.Fatalf("project deletion did not prepare automatically: %v", err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(context.Background(), false); err == nil {
		t.Fatal("changed deletion target executed")
	}
	for _, directory := range []string{inside, outside} {
		if _, err := os.Stat(filepath.Join(directory, "file")); err != nil {
			t.Fatal("stale deletion changed a file:", err)
		}
	}
}

func TestDeletionPolicyFollowsIntermediateLinks(t *testing.T) {
	e, project := testExecutor(t)
	outside := t.TempDir()
	path := filepath.Join(outside, "file")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, "outside-link")); err != nil {
		t.Fatal(err)
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"rm outside-link/file"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !op.Approval().Required {
		t.Fatal("external target behind a link was treated as a project child")
	}
	if _, err := op.Execute(context.Background(), false); err == nil {
		t.Fatal("external deletion ran without approval")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("unapproved external deletion changed its target:", err)
	}
	if err := os.Symlink(project, filepath.Join(project, "root-link")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"rm -rf root-link/"}`)); err == nil {
		t.Fatal("project root reached through a link was not blocked")
	}
}
