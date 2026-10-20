// Purpose: Verify scoped file tools, approval binding and stale-change rejection.
// Pattern: Filesystem behavior tests.
// Usage: go test ./internal/tools/local.
// Related: internal/tools/local/executor.go, internal/tools/local/textedit/edit.go, internal/policy/policy.go.

package local

import (
	"context"
	"encoding/json"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testExecutor(t *testing.T) (*Executor, string) {
	t.Helper()
	project, home := t.TempDir(), t.TempDir()
	executor, err := NewExecutor(project, home, filepath.Join(home, ".v"), func(text string) string { return strings.ReplaceAll(text, "provider-secret", "[redacted]") })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	})
	return executor, project
}

func call(name, arguments string) toolapi.Call {
	return toolapi.Call{ID: "call_1", Name: name, Arguments: json.RawMessage(arguments)}
}

func TestExternalWriteRequiresApprovalAndPreservesReject(t *testing.T) {
	executor, _ := testExecutor(t)
	path := filepath.Join(t.TempDir(), "auth.go")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.System}
	body, err := json.Marshal(map[string]string{"path": path, "content": "new"})
	if err != nil {
		t.Fatal(err)
	}
	operation, err := executor.Prepare(context.Background(), settings, call("write_file", string(body)))
	if err != nil {
		t.Fatal(err)
	}
	if !operation.Approval().Required || !strings.Contains(operation.Approval().Preview, "new") {
		t.Fatalf("missing preview: %+v", operation.Approval())
	}
	if _, err := operation.Execute(context.Background(), false); err == nil {
		t.Fatal("unapproved write ran")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "old" {
		t.Fatalf("rejected write changed file: %q, %v", content, err)
	}
	if _, err := operation.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(path)
	if err != nil || string(content) != "new" {
		t.Fatalf("approved write: %q, %v", content, err)
	}
	if _, err := operation.Execute(context.Background(), true); err == nil {
		t.Fatal("replayed a prepared mutation")
	}
}
func TestApprovalUsesLineDiff(t *testing.T) {
	executor, project := testExecutor(t)
	path := filepath.Join(project, "file.txt")
	if err := os.WriteFile(path, []byte("unchanged\nold\nend\n"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.Directory}
	for _, invocation := range []toolapi.Call{
		call("write_file", `{"path":"file.txt","content":"unchanged\nnew\nend\n"}`),
		observedEdit(t, executor, "file.txt", 2, 2, "new\n"),
	} {
		op, err := executor.Prepare(context.Background(), settings, invocation)
		if err != nil {
			t.Fatal(err)
		}
		preview := op.Approval().Preview
		if !op.Approval().Diff || !strings.Contains(preview, "@@ -") || !strings.Contains(preview, "-old\n") {
			t.Fatalf("missing diff for %s: %s", invocation.Name, preview)
		}
		if !strings.Contains(preview, "+new\n") {
			t.Fatal("replacement omitted from approval")
		}
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "unchanged\nold\nend\n" {
		t.Fatal("generating an approval preview changed the file")
	}
}

func TestApprovedEditRejectsChangedOriginal(t *testing.T) {
	executor, project := testExecutor(t)
	path := filepath.Join(project, "auth.go")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	operation, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.Directory}, observedEdit(t, executor, "auth.go", 1, 1, "fixed"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("external change"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := operation.Execute(context.Background(), true); err == nil {
		t.Fatal("overwrote a stale version")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "external change" {
		t.Fatalf("lost external change: %q, %v", content, err)
	}
}

func TestFileToolsRejectEscapesSecretsAndInvalidArguments(t *testing.T) {
	executor, project := testExecutor(t)
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, "linked")); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.Directory}
	for _, invocation := range []toolapi.Call{
		call("write_file", `{"path":"../outside","content":"unsafe"}`),
		call("write_file", `{"path":"linked","content":"unsafe"}`),
		call("write_file", `{"path":"auth.go","content":"new","approved":true}`),
		call("write_file", `{"path":"auth.go","content":"provider-secret"}`),
	} {
		if _, err := executor.Prepare(context.Background(), settings, invocation); err == nil {
			t.Errorf("accepted unsafe call: %+v", invocation)
		}
	}

}

func TestReadDirectorySearchAndMove(t *testing.T) {
	executor, project := testExecutor(t)
	if err := os.Mkdir(filepath.Join(project, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "src/auth.go"), []byte("func authenticate() {}\nprovider-secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.System}
	for _, invocation := range []toolapi.Call{
		call("execute_command", `{"command":"cat src/auth.go"}`),
		call("execute_command", `{"command":"ls src"}`),
		call("execute_command", `{"command":"find . -name '*.go'"}`),
		call("execute_command", `{"command":"grep -rn authenticate src"}`),
	} {
		operation, err := executor.Prepare(context.Background(), settings, invocation)
		if err != nil {
			t.Fatal(err)
		}
		result, err := operation.Execute(context.Background(), false)
		if err != nil || !json.Valid(result.Content) || strings.Contains(string(result.Content), "provider-secret") {
			t.Fatalf("tool result: %s, %v", result.Content, err)
		}
	}
	operation, err := executor.Prepare(context.Background(), settings, call("move_file", `{"path":"src/auth.go","destination":"src/session.go"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := operation.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "src/session.go")); err != nil {
		t.Fatal(err)
	}
}
