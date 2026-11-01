// Purpose: Verify the LAT capability boundary, ordinary observations and main-executor independence.
// Pattern: Native tool integration tests.
// Usage: go test ./internal/tools/local -run TestReadOnly.
// Related: internal/tools/local/read_only.go, internal/tools/local/read_only_command.go.
package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func readOnlyForTest(t *testing.T, parent *Executor) *readOnlyExecutor {
	t.Helper()
	child, close, err := parent.ReadOnly()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := close(); err != nil {
			t.Error(err)
		}
	})
	return child.(*readOnlyExecutor)
}

func defaultReadOnlySettings() policy.Settings { return policy.Settings{Scope: policy.System} }

func readOnlyQuery(t *testing.T, executor toolapi.Executor, command string) toolapi.Execution {
	t.Helper()
	args, err := json.Marshal(map[string]string{"command": command})
	if err != nil {
		t.Fatal(err)
	}
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.PermissionAllMighty}, call("execute_command", string(args)))
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func TestReadOnlyToolsAndObservations(t *testing.T) {
	parent, project := testExecutor(t)
	child := readOnlyForTest(t, parent)
	want := map[string]bool{"execute_command": true, "read_image": true, "list_command_runs": true, "wait_command": true, "cancel_command": true}
	for _, definition := range child.Definitions() {
		if !want[definition.Name] {
			t.Fatalf("unexpected tool %s", definition.Name)
		}
		delete(want, definition.Name)
		if definition.Name == "execute_command" && strings.Contains(definition.Description, "builds, tests, package management") {
			t.Fatal("inherited unrestricted description")
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing tools: %v", want)
	}
	if err := os.WriteFile(filepath.Join(project, "readme"), []byte("first\nprovider-secret\nlast\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for command, text := range map[string]string{
		"cat readme | head -n 1": "first", "sed -n '2,2p' readme": "[redacted]", "find . -name readme -print && wc -l readme": "readme", "grep -n last readme": "last",
	} {
		result, err := readOnlyQuery(t, child, command).Execute(context.Background(), true)
		if err != nil || result.Failed || !strings.Contains(string(result.Content), text) || strings.Contains(string(result.Content), "provider-secret") {
			t.Fatalf("%s: %+v %v", command, result, err)
		}
	}
	for _, name := range []string{"write_file", "edit_file", "create_directory"} {
		if _, err := child.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: policy.PermissionAllMighty}, call(name, `{}`)); err == nil {
			t.Fatalf("exposed %s", name)
		}
	}
	if _, err := child.Prepare(context.Background(), policy.Settings{Scope: policy.Directory}, call("execute_command", `{"command":"cat readme"}`)); err == nil {
		t.Fatal("claimed directory confinement")
	}
}

func TestReadOnlyRechecksAtExecutionAndPreservesParent(t *testing.T) {
	parent, project := testExecutor(t)
	child := readOnlyForTest(t, parent)
	for _, command := range []string{"printf changed > changed", "rm readme", "git diff --output=changed", "rg --pre sh value"} {
		op := readOnlyQuery(t, child, "pwd").(*readOnlyOperation)
		// Exercise the execution check separately from preparation and permission.
		op.command.input.Command = command
		if _, err := op.Execute(context.Background(), true); err == nil {
			t.Fatalf("execution bypassed boundary: %s", command)
		}
	}
	op := readOnlyQuery(t, child, "pwd").(*readOnlyOperation)
	op.command.settings.Scope = policy.Directory
	if _, err := op.Execute(context.Background(), true); err == nil {
		t.Fatal("execution claimed directory confinement")
	}
	credential := readOnlyQuery(t, child, "pwd").(*readOnlyOperation)
	credential.command.input.Command = "printf provider-secret"
	if _, err := credential.Execute(context.Background(), true); err == nil {
		t.Fatal("execution exposed a protected credential")
	}
	if _, err := os.Stat(filepath.Join(project, "changed")); !os.IsNotExist(err) {
		t.Fatal("read-only command wrote a file")
	}
	result, err := readOnlyQuery(t, parent, "printf parent > changed").Execute(context.Background(), true)
	if err != nil || result.Failed {
		t.Fatalf("main executor restricted: %+v %v", result, err)
	}
	if content, err := os.ReadFile(filepath.Join(project, "changed")); err != nil || string(content) != "parent" {
		t.Fatalf("parent command: %q %v", content, err)
	}
}

func TestReadOnlyScrubsEnvironmentAndLookup(t *testing.T) {
	parent, project := testExecutor(t)
	child := readOnlyForTest(t, parent)
	malicious := filepath.Join(project, "bin")
	if err := os.Mkdir(malicious, 0700); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf hacked > '" + filepath.Join(project, "changed") + "'\n"
	for _, name := range []string{"cat", "git"} {
		if err := os.WriteFile(filepath.Join(malicious, name), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
	}
	hook := filepath.Join(project, "startup")
	if err := os.WriteFile(hook, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", malicious)
	t.Setenv("ENV", hook)
	t.Setenv("BASH_ENV", hook)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.fsmonitor")
	t.Setenv("GIT_CONFIG_VALUE_0", hook)
	if err := os.WriteFile(filepath.Join(project, "readme"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := readOnlyQuery(t, child, "cat readme").Execute(context.Background(), false)
	if err != nil || !strings.Contains(string(result.Content), "original") {
		t.Fatalf("query lookup: %+v %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(project, "changed")); !os.IsNotExist(err) {
		t.Fatal("inherited executable hook ran")
	}
}
