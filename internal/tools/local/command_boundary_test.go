//go:build linux || darwin

// Purpose: Verify real shell deletion, external write denial, partial effects and trusted exceptions.
// Pattern: Native tool integration tests.
// Usage: go test ./internal/tools/local -run TestShellBoundary.
// Related: internal/tools/local/command.go, internal/tools/local/filesystem/boundary.go.
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

func TestShellBoundaryDeletionAndPartialFailure(t *testing.T) {
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
	for _, d := range e.Definitions() {
		if d.Name == "delete_file" {
			t.Fatal("removed tool advertised")
		}
	}
	command := fmt.Sprintf("rm -rf binary.db nested; rm %q", outside)
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.Approve}, call("execute_command", fmt.Sprintf(`{"command":%q}`, command)))
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
	if err != nil || !out.Failed || !out.Started || out.Process.ExitCode == 0 {
		t.Fatalf("partial failure lost: %+v %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(project, "binary.db")); !os.IsNotExist(err) {
		t.Fatal("local file remains")
	}
	if _, err := os.Stat(filepath.Join(project, "nested")); !os.IsNotExist(err) {
		t.Fatal("recursive removal failed")
	}
	if content, err := os.ReadFile(outside); err != nil || len(content) != 3 {
		t.Fatal("external file changed")
	}
}

func TestShellBoundaryAdditionalWriteRoot(t *testing.T) {
	project, home, cache := t.TempDir(), t.TempDir(), t.TempDir()
	e, err := NewExecutor(project, home, filepath.Join(home, ".v"), []string{cache}, func(s string) string { return s })
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := e.Close(); err != nil {
			t.Error(err)
		}
	}()
	canonical, err := filepath.EvalSymlinks(cache)
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range e.Definitions() {
		if definition.Name == "execute_command" && !strings.Contains(definition.Description, canonical) {
			t.Fatal("model did not receive trusted write roots")
		}
	}
	path := filepath.Join(cache, "file")
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.AllowAll}, call("write_file", fmt.Sprintf(`{"path":%q,"content":"test"}`, path)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	op, err = e.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.AllowAll}, call("execute_command", fmt.Sprintf(`{"command":%q}`, "rm "+fmt.Sprintf("%q", path))))
	if err != nil {
		t.Fatal(err)
	}
	out, err := op.Execute(context.Background(), false)
	if err != nil || out.Failed {
		t.Fatalf("exception not usable: %+v %v", out, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("exception file remains")
	}
	outside := filepath.Join(home, "blocked")
	if _, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.AllowAll}, call("write_file", fmt.Sprintf(`{"path":%q,"content":"escape"}`, outside))); err == nil {
		t.Fatal("file tool escaped boundary")
	}
}
