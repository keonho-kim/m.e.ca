//go:build linux || darwin

// Purpose: Verify host file operations retain approval and can reach paths outside the project.
// Pattern: Native tool integration tests.
// Usage: go test ./internal/tools/local -run TestHostFileOperations.
// Related: internal/tools/local/command.go, internal/tools/local/filesystem/boundary.go.
package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestHostFileOperationsDeletionRequiresApproval(t *testing.T) {
	e, project := testExecutor(t)
	outside := filepath.Join(t.TempDir(), "safe.db")
	for _, p := range []string{outside, filepath.Join(project, "binary.db")} {
		if err := os.WriteFile(p, []byte{0, 1, 2}, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(project, "nested/deep"), 0700); err != nil {
		t.Fatal(err)
	}
	command := fmt.Sprintf("rm -rf binary.db nested; rm %q", outside)
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, command)))
	if err != nil {
		t.Fatal(err)
	}
	if !op.Approval().Required {
		t.Fatal("missing approval")
	}
	if _, err := op.Execute(context.Background(), false); err == nil {
		t.Fatal("unapproved deletion")
	}
	out, err := op.Execute(context.Background(), true)
	if err != nil || out.Failed || !out.Started || out.Process.ExitCode != 0 {
		t.Fatalf("host deletion failed: %+v %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(project, "binary.db")); !os.IsNotExist(err) {
		t.Fatal("local file remains")
	}
	if _, err := os.Stat(filepath.Join(project, "nested")); !os.IsNotExist(err) {
		t.Fatal("recursive removal failed")
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatal("approved external deletion did not run")
	}
}

func TestHostFileOperationsAbsoluteFileAndShellPaths(t *testing.T) {
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	e, err := NewExecutor(project, home, filepath.Join(home, ".meca"), func(s string) string { return s })
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := e.Close(); err != nil {
			t.Error(err)
		}
	}()
	path := filepath.Join(cache, "file")
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("write_file", fmt.Sprintf(`{"path":%q,"content":"test"}`, path)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	op, err = e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, "rm "+fmt.Sprintf("%q", path))))
	if err != nil {
		t.Fatal(err)
	}
	out, err := op.Execute(context.Background(), true)
	if err != nil || out.Failed {
		t.Fatalf("host path not usable: %+v %v", out, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("host file remains")
	}
}
