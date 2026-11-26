// Purpose: Verify development search through the CLI and the agent's default command policy.
// Pattern: CLI and native-tool integration test with isolated installed state.
// Usage: go test ./cmd/meca -run TestDevelopExecutable.
// Related: cmd/meca/main.go, internal/commands/develop/search.go, internal/tools/local/command_operation.go.
package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local"
)

func TestDevelopExecutable(t *testing.T) {
	binary := testExecutable(t)
	home := t.TempDir()
	run := func(program string, args ...string) (string, error) {
		command := exec.CommandContext(t.Context(), program, args...)
		command.Env = append(os.Environ(), "HOME="+home, "SHELL=/bin/sh", "PATH=/usr/bin:/bin")
		output, err := command.CombinedOutput()
		return string(output), err
	}
	if output, err := run(binary, "develop", "--help"); err != nil || !strings.Contains(output, "meca develop") {
		t.Fatalf("develop help: %v\n%s", err, output)
	}
	if output, err := run(binary, "develop", "queue"); err == nil || !strings.Contains(output, "meca install") {
		t.Fatalf("missing installation: %v\n%s", err, output)
	}
	if output, err := run(binary, "install", "--no-shell"); err != nil {
		t.Fatalf("install: %v\n%s", err, output)
	}
	installed := filepath.Join(home, ".local/bin/meca")
	if output, err := run(installed, "develop", "delayed", "tasks"); err != nil || !strings.Contains(output, "Delayed tasks and expiring entries") {
		t.Fatalf("installed search: %v\n%s", err, output)
	}
	if output, err := run(installed, "develop"); err == nil || !strings.Contains(output, "usage: meca develop") {
		t.Fatalf("empty search: %v\n%s", err, output)
	}
	checkDevelopAgentCommand(t, home)
}

func checkDevelopAgentCommand(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("PATH", filepath.Join(home, ".local/bin")+":/usr/bin:/bin")
	for _, name := range []string{"ENV", "BASH_ENV", "SHELLOPTS", "BASHOPTS", "LD_PRELOAD", "DYLD_INSERT_LIBRARIES"} {
		t.Setenv(name, "")
	}
	// A running agent holds shared leases on both directories.
	for _, directory := range []string{home, filepath.Join(home, ".meca")} {
		root, err := os.OpenRoot(directory)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { root.Close() })
		lease, err := statefs.Lease(root, false)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { lease.Close() })
	}
	// The interpreter can read the private script without an executable bit.
	script := filepath.Join(home, ".meca/skills/development/scripts/search.sh")
	if err := os.Chmod(script, 0400); err != nil {
		t.Fatal(err)
	}
	executor, err := local.NewExecutor(t.TempDir(), home, filepath.Join(home, ".meca"), func(s string) string { return s }, local.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	})
	executor.BeginCommands(t.Context(), time.Now().Add(time.Minute))
	settings := policy.Settings{Scope: policy.System, Permission: policy.PermissionDefault}
	execute := func(name string, args map[string]any) toolapi.Result {
		t.Helper()
		body, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		operation, err := executor.Prepare(t.Context(), settings, toolapi.Call{ID: "develop-" + name, Name: name, Arguments: body})
		if err != nil {
			t.Fatal(err)
		}
		defer toolapi.Release(operation)
		if proposal := operation.Approval(); proposal.Required || proposal.UserRequired {
			t.Fatalf("guide search requested additional approval: %+v", proposal)
		}
		result, err := operation.Execute(t.Context(), false)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	result := execute("execute_command", map[string]any{
		"reason": "Find development guidance", "command": `meca develop "delayed task queue"`,
	})
	for result.Process != nil && toolapi.CommandActive(result.Process.Status) {
		result = execute("wait_command", map[string]any{"command_id": result.Process.CommandID, "wait_seconds": 1})
	}
	if result.Failed || result.Process == nil || result.Process.ExitCode != 0 || !strings.Contains(string(result.Content), "Delayed tasks and expiring entries") {
		t.Fatalf("agent guide search failed: %+v\n%s", result.Process, result.Content)
	}
}
