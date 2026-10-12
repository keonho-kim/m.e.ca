// Purpose: Verify scoped file tools, approval binding and stale-change rejection.
// Pattern: Filesystem behavior tests.
// Usage: go test ./internal/tools/local.
// Related: internal/tools/local/executor.go, internal/tools/local/files.go, internal/policy/policy.go.

package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/policy"
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

func call(name, arguments string) agent.ToolCall {
	return agent.ToolCall{ID: "call_1", Name: name, Arguments: json.RawMessage(arguments)}
}

func TestWriteRequiresApprovalAndPreservesReject(t *testing.T) {
	executor, project := testExecutor(t)
	path := filepath.Join(project, "auth.go")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Permission: policy.Approve, Scope: policy.Directory}
	operation, err := executor.Prepare(context.Background(), settings, false, call("write_file", `{"path":"auth.go","content":"new"}`))
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

func TestApprovedEditRejectsChangedOriginal(t *testing.T) {
	executor, project := testExecutor(t)
	path := filepath.Join(project, "auth.go")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	operation, err := executor.Prepare(context.Background(), policy.Settings{Permission: policy.Approve, Scope: policy.Directory}, false, call("edit_file", `{"path":"auth.go","old_text":"original","new_text":"fixed"}`))
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
	settings := policy.Settings{Permission: policy.AllowAll, Scope: policy.Directory}
	for _, invocation := range []agent.ToolCall{
		call("read_file", `{"path":"../outside"}`),
		call("read_file", `{"path":"linked"}`),
		call("read_file", `{"path":"auth.go","approved":true}`),
		call("write_file", `{"path":"auth.go","content":"provider-secret"}`),
		call("execute_command", `{"command":"touch outside"}`),
	} {
		if _, err := executor.Prepare(context.Background(), settings, false, invocation); err == nil {
			t.Errorf("accepted unsafe call: %+v", invocation)
		}
	}
	if _, err := executor.Prepare(context.Background(), settings, true, call("write_file", `{"path":"new","content":"blocked"}`)); err == nil {
		t.Fatal("planning changed a file")
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
	settings := policy.Settings{Permission: policy.AllowAll, Scope: policy.Directory}
	for _, invocation := range []agent.ToolCall{
		call("read_file", `{"path":"src/auth.go"}`),
		call("list_directory", `{"path":"src"}`),
		call("search_files", `{"path":".","pattern":"*.go"}`),
		call("grep", `{"path":"src","pattern":"authenticate"}`),
	} {
		operation, err := executor.Prepare(context.Background(), settings, false, invocation)
		if err != nil {
			t.Fatal(err)
		}
		result, err := operation.Execute(context.Background(), false)
		if err != nil || !json.Valid(result.Content) || strings.Contains(string(result.Content), "provider-secret") {
			t.Fatalf("tool result: %s, %v", result.Content, err)
		}
	}
	operation, err := executor.Prepare(context.Background(), settings, false, call("move_file", `{"path":"src/auth.go","destination":"src/session.go"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := operation.Execute(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "src/session.go")); err != nil {
		t.Fatal(err)
	}
}
