// Purpose: Verify shell approvals bind target identities and the actual execution environment.
// Pattern: Native policy integration tests.
// Usage: go test ./internal/tools/local -run 'TestCommandBinding|TestShellPath|TestMutationSafety'.
// Related: internal/tools/local/command_safety.go, internal/tools/local/shell_paths.go.
package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/policy"
)

func TestCommandBindingFreezesEnvironment(t *testing.T) {
	e, project := testExecutor(t)
	t.Setenv("V_TEST_VALUE", "before")
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"printf '%s' \"$V_TEST_VALUE\" > value"}`))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("V_TEST_VALUE", "after")
	if result, err := op.Execute(context.Background(), false); err != nil || result.Failed {
		t.Fatalf("%+v %v", result, err)
	}
	if b, err := os.ReadFile(filepath.Join(project, "value")); err != nil || string(b) != "before" {
		t.Fatalf("%q %v", b, err)
	}
}

func TestCommandBindingRejectsChangedAlias(t *testing.T) {
	e, project := testExecutor(t)
	outside := t.TempDir()
	inside := filepath.Join(project, "inside")
	if err := os.Mkdir(inside, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(project, "alias")
	if err := os.Symlink(inside, alias); err != nil {
		t.Fatal(err)
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"printf changed > alias/file"}`))
	if err != nil {
		t.Fatal(err)
	}
	if op.Approval().Required {
		t.Fatal("ordinary project write required approval")
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(context.Background(), true); err == nil {
		t.Fatal("changed target reused approval")
	}
	if _, err := os.Stat(filepath.Join(outside, "file")); !os.IsNotExist(err) {
		t.Fatal("changed target executed")
	}
}

func TestShellPathUnlinksAliasWithoutFollowingFinalLink(t *testing.T) {
	e, project := testExecutor(t)
	alias := filepath.Join(project, "system-link")
	if err := os.Symlink("/etc", alias); err != nil {
		t.Fatal(err)
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"rm system-link"}`))
	if err != nil {
		t.Fatal(err)
	}
	if op.Approval().Required {
		t.Fatal("removing a project-local link followed its external destination")
	}
	if result, err := op.Execute(context.Background(), false); err != nil || result.Failed {
		t.Fatalf("%+v %v", result, err)
	}
	if _, err := os.Lstat(alias); !os.IsNotExist(err) {
		t.Fatal("link remains")
	}
}

func TestShellPathPreservesSymlinkParentTraversal(t *testing.T) {
	e, project := testExecutor(t)
	outside := t.TempDir()
	child := filepath.Join(outside, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(child, filepath.Join(project, "link")); err != nil {
		t.Fatal(err)
	}
	result, err := e.analyzeCommand(context.Background(), "printf hi > link/../target", os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.decision.Reasons) == 0 || !strings.Contains(result.decision.Description(), filepath.Join(canonical, "target")) {
		t.Fatalf("wrong traversal: %+v", result.decision)
	}
}

func TestMutationSafetyOutsideProject(t *testing.T) {
	e, _ := testExecutor(t)
	outside := filepath.Join(t.TempDir(), "new")
	for _, invocation := range []struct{ name, args string }{
		{"write_file", fmt.Sprintf(`{"path":%q,"content":"value"}`, outside)},
		{"create_directory", fmt.Sprintf(`{"paths":[%q]}`, outside)},
	} {
		op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call(invocation.name, invocation.args))
		if err != nil {
			t.Fatal(err)
		}
		if !op.Approval().Required {
			t.Fatal("external change bypassed safety approval")
		}
		if _, err := op.Execute(context.Background(), false); err == nil {
			t.Fatal("external change executed without approval")
		}
		if _, err := os.Stat(outside); !os.IsNotExist(err) {
			t.Fatal("unapproved path exists")
		}
	}
}
