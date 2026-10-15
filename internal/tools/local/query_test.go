// Purpose: Verify Plan search queries remain read-only and preserve scoped, bounded results.
// Pattern: Filesystem integration tests.
// Usage: go test ./internal/tools/local -run TestPlanQuery.
// Related: internal/tools/local/query.go, internal/policy/shell/query.go.
package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/tools/local/process"
)

func TestPlanQueryResults(t *testing.T) {
	executor, project := testExecutor(t)
	if err := os.Mkdir(filepath.Join(project, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "src", "a.go"), []byte("TODO provider-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.System, Permission: policy.Approve}
	for _, tc := range []struct{ command, want string }{
		{"ls -la src", "a.go"}, {"find . -type f -name '*.go'", "a.go"}, {"rg --files -g '*.go' .", "a.go"}, {"rg -n TODO src", "a.go:1:TODO [redacted]"}, {"grep -rn TODO src", "a.go:1:TODO [redacted]"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			arguments, _ := json.Marshal(map[string]string{"command": tc.command})
			op, err := executor.Prepare(context.Background(), settings, true, call("execute_command", string(arguments)))
			if err != nil {
				t.Fatal(err)
			}
			if op.Approval().Required {
				t.Fatal("read-only Plan query requested write approval")
			}
			result, err := op.Execute(context.Background(), false)
			if err != nil {
				t.Fatal(err)
			}
			var output process.Output
			if err := json.Unmarshal(result.Content, &output); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.Stdout, tc.want) || strings.Contains(output.Stdout, "provider-secret") || output.Status != "query_completed" || result.Process != nil {
				t.Fatalf("query result=%+v process=%+v", output, result.Process)
			}
		})
	}
	// External programs named like supported commands must never be started in Plan.
	t.Setenv("PATH", project)
	if err := os.WriteFile(filepath.Join(project, "ls"), []byte("#!/bin/sh\ntouch invaded\n"), 0700); err != nil {
		t.Fatal(err)
	}
	op, err := executor.Prepare(context.Background(), settings, true, call("execute_command", `{"command":"ls"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "invaded")); !os.IsNotExist(err) {
		t.Fatal("external command executed")
	}
}

func TestPlanQueryRejectsMutationsAndEscapes(t *testing.T) {
	executor, project := testExecutor(t)
	settings := policy.Settings{Scope: policy.Directory, Permission: policy.AllowAll}
	for _, command := range []string{"find . -delete", "find . -exec rm x ';'", "find . -fprint x", "rg --pre=sh x", "ls > x", "ls && touch x", "ls $(touch x)", "sh -c ls", "find .."} {
		args, _ := json.Marshal(map[string]string{"command": command})
		if op, err := executor.Prepare(context.Background(), settings, true, call("execute_command", string(args))); err == nil || op != nil {
			t.Errorf("Plan command accepted: %s", command)
		}
	}
	for _, name := range []string{"list_directory", "search_files", "grep"} {
		if _, err := executor.Prepare(context.Background(), settings, false, call(name, `{"path":"."}`)); err == nil {
			t.Errorf("removed tool accepted: %s", name)
		}
	}
	for _, def := range executor.Definitions() {
		if def.Name == "list_directory" || def.Name == "search_files" || def.Name == "grep" {
			t.Fatal("removed search tool advertised")
		}
	}
	if _, err := os.Stat(filepath.Join(project, "x")); !os.IsNotExist(err) {
		t.Fatal("blocked query wrote a file")
	}
}

func TestPlanQueryDepthAndCancellation(t *testing.T) {
	executor, project := testExecutor(t)
	if err := os.MkdirAll(filepath.Join(project, "a", "b"), 0700); err != nil {
		t.Fatal(err)
	}
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.Directory, Permission: policy.AllowAll}, true, call("execute_command", `{"command":"find . -type d -maxdepth 1"}`))
	if err != nil {
		t.Fatal(err)
	}
	result, err := op.Execute(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(result.Content), "/a/b") {
		t.Fatal("maxdepth ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := executor.Prepare(ctx, policy.Settings{Scope: policy.System, Permission: policy.AllowAll}, true, call("execute_command", `{"command":"ls"}`)); err == nil {
		t.Fatal("cancelled query prepared")
	}
}

func TestNormalSearchCommandsUseProcess(t *testing.T) {
	executor, project := testExecutor(t)
	if err := os.WriteFile(filepath.Join(project, "a.txt"), []byte("needle\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"find . -name '*.txt' | head -n 2", "ls -1", "grep -n needle a.txt"} {
		args, _ := json.Marshal(map[string]string{"command": command})
		op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.Approve}, false, call("execute_command", string(args)))
		if err != nil {
			t.Fatal(err)
		}
		if !op.Approval().Required {
			t.Fatal("shell approval bypassed")
		}
		result, err := op.Execute(context.Background(), true)
		if err != nil || result.Failed || result.Process == nil || result.Process.ExitCode != 0 {
			t.Fatalf("%s: %+v %v", command, result, err)
		}
	}
}

func TestPlanQuerySkipsSymlinksAndPrivateState(t *testing.T) {
	executor, project := testExecutor(t)
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("PRIVATE-MARKER"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, "linked")); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.Directory, Permission: policy.AllowAll}
	op, err := executor.Prepare(context.Background(), settings, true, call("execute_command", `{"command":"rg -n PRIVATE-MARKER ."}`))
	if err != nil {
		t.Fatal(err)
	}
	result, err := op.Execute(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	var output process.Output
	if err := json.Unmarshal(result.Content, &output); err != nil {
		t.Fatal(err)
	}
	if output.Stdout != "" {
		t.Fatalf("symlink content exposed: %s", output.Stdout)
	}
	args, _ := json.Marshal(map[string]string{"command": "find '" + executor.private + "'"})
	if _, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.AllowAll}, true, call("execute_command", string(args))); err == nil {
		t.Fatal("private state query accepted")
	}
}
