// Purpose: Verify automatic internal moves preserve boundary approval, symlink semantics and stale-target rejection.
// Pattern: Temporary-filesystem integration tests.
// Usage: go test ./internal/tools/local -run TestMovePolicy.
// Related: internal/tools/local/command_safety.go, internal/tools/local/mutation_safety.go, internal/policy/command/move.go.
package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func TestMovePolicyInternalShellMoves(t *testing.T) {
	type moveCase struct {
		command, removed string
		expected         map[string]string
	}
	tests := []moveCase{
		{`mv one.txt renamed.txt`, "one.txt", map[string]string{"renamed.txt": "one"}},
		{`mv nested/three.txt .`, "nested/three.txt", map[string]string{"three.txt": "three"}},
		{`mv -fn one.txt renamed.txt`, "one.txt", map[string]string{"renamed.txt": "one"}},
		{`mv one.txt two.txt out`, "one.txt", map[string]string{"out/one.txt": "one", "out/two.txt": "two"}},
		{`mv one.txt two.txt`, "one.txt", map[string]string{"two.txt": "one"}},
		{`mv nested renamed`, "nested", map[string]string{"renamed/three.txt": "three"}},
		{`cd nested; mv three.txt renamed.txt`, "nested/three.txt", map[string]string{"nested/renamed.txt": "three"}},
		{`mv "spaced name.txt" "renamed name.txt"`, "spaced name.txt", map[string]string{"renamed name.txt": "space"}},
	}
	if runtime.GOOS == "linux" {
		tests = append(tests, moveCase{`mv -t out one.txt two.txt`, "one.txt", map[string]string{"out/one.txt": "one", "out/two.txt": "two"}})
	}
	for _, test := range tests {
		t.Run(test.command, func(t *testing.T) {
			e, project := testExecutor(t)
			for _, directory := range []string{"nested", "out"} {
				if err := os.Mkdir(filepath.Join(project, directory), 0700); err != nil {
					t.Fatal(err)
				}
			}
			for name, body := range map[string]string{"one.txt": "one", "two.txt": "two", "nested/three.txt": "three", "spaced name.txt": "space"} {
				if err := os.WriteFile(filepath.Join(project, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, test.command)))
			if err != nil || op.Approval().Required {
				t.Fatalf("internal move did not prepare automatically: %v", err)
			}
			if result, err := op.Execute(context.Background(), false); err != nil || result.Failed {
				t.Fatalf("internal move failed: %+v %v", result, err)
			}
			for name, want := range test.expected {
				if body, err := os.ReadFile(filepath.Join(project, name)); err != nil || string(body) != want {
					t.Fatalf("%s=%q; want %q: %v", name, body, want, err)
				}
			}
			if _, err := os.Lstat(filepath.Join(project, test.removed)); !os.IsNotExist(err) {
				t.Fatalf("moved source remains: %v", err)
			}
		})
	}
}

func TestMovePolicyExternalEndpointsNeedApproval(t *testing.T) {
	for _, name := range []string{"execute_command", "move_file"} {
		for _, inbound := range []bool{false, true} {
			e, project := testExecutor(t)
			source, destination := filepath.Join(project, "source"), filepath.Join(t.TempDir(), "destination")
			if inbound {
				source, destination = destination, source
			}
			if err := os.WriteFile(source, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			args := fmt.Sprintf(`{"path":%q,"destination":%q}`, source, destination)
			if name == "execute_command" {
				args = fmt.Sprintf(`{"command":%q}`, fmt.Sprintf("mv %q %q", source, destination))
			}
			op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call(name, args))
			if err != nil || !op.Approval().Required {
				t.Fatalf("%s inbound=%v: external move was not reviewed: %v", name, inbound, err)
			}
			if _, err := op.Execute(context.Background(), false); err == nil {
				t.Fatal("unapproved move executed")
			}
			if body, err := os.ReadFile(source); err != nil || string(body) != "keep" {
				t.Fatalf("denied move changed source: %q %v", body, err)
			}
			if _, err := os.Lstat(destination); !os.IsNotExist(err) {
				t.Fatalf("denied move created destination: %v", err)
			}
		}
	}
}

func TestMovePolicyUnresolvedShellOperandsNeedApproval(t *testing.T) {
	e, project := testExecutor(t)
	if err := os.Mkdir(filepath.Join(project, "out"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "source.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{
		`mv *.txt out`,
		fmt.Sprintf(`mv %q/*.txt out`, project),
		`read -r paths; mv $paths`,
		`printf 'source.txt out' | xargs mv`,
	} {
		op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, command)))
		if err != nil || !op.Approval().Required {
			t.Fatalf("unresolved move did not require approval: %s: %v", command, err)
		}
		if _, err := op.Execute(context.Background(), false); err == nil {
			t.Fatalf("unapproved dynamic move executed: %s", command)
		}
	}
	if body, err := os.ReadFile(filepath.Join(project, "source.txt")); err != nil || string(body) != "keep" {
		t.Fatalf("denied dynamic move changed source: %q %v", body, err)
	}
}

func TestMovePolicyDestinationLinks(t *testing.T) {
	for _, directory := range []bool{false, true} {
		e, project := testExecutor(t)
		external := filepath.Join(t.TempDir(), "target")
		if directory {
			if err := os.Mkdir(external, 0700); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(external, []byte("untouched"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, "source"), []byte("moved"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(external, filepath.Join(project, "alias")); err != nil {
			t.Fatal(err)
		}
		op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"mv source alias"}`))
		if err != nil || op.Approval().Required != directory {
			t.Fatalf("directory=%v: wrong symlink move decision: %v", directory, err)
		}
		result, err := op.Execute(context.Background(), false)
		if directory {
			if err == nil {
				t.Fatal("external directory move ran without approval")
			}
			if _, err := os.Stat(filepath.Join(external, "source")); !os.IsNotExist(err) {
				t.Fatalf("external directory changed: %v", err)
			}
		} else {
			if err != nil || result.Failed {
				t.Fatalf("replacing a local file symlink failed: %+v %v", result, err)
			}
			if body, err := os.ReadFile(external); err != nil || string(body) != "untouched" {
				t.Fatalf("symlink target was modified: %q %v", body, err)
			}
			if body, err := os.ReadFile(filepath.Join(project, "alias")); err != nil || string(body) != "moved" {
				t.Fatalf("wrong replacement entry: %q %v", body, err)
			}
		}
	}
}

func TestMovePolicyRechecksDirectoryAlias(t *testing.T) {
	e, project := testExecutor(t)
	inside, outside := filepath.Join(project, "inside"), t.TempDir()
	if err := os.Mkdir(inside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "source"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(project, "alias")
	if err := os.Symlink(inside, alias); err != nil {
		t.Fatal(err)
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"mv source alias"}`))
	if err != nil || op.Approval().Required {
		t.Fatalf("internal alias move required approval: %v", err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, alias); err != nil {
		t.Fatal(err)
	}
	_, err = op.Execute(context.Background(), false)
	var failure *toolapi.Failure
	if !errors.As(err, &failure) || failure.Code != toolapi.StaleTarget {
		t.Fatalf("changed alias executed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "source")); err != nil {
		t.Fatal("stale move changed source:", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "source")); !os.IsNotExist(err) {
		t.Fatal("stale move wrote outside the project:", err)
	}
}

func TestMovePolicyBlocksProtectedDirectoryAlias(t *testing.T) {
	e, project := testExecutor(t)
	if err := os.WriteFile(filepath.Join(project, "source"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc", filepath.Join(project, "alias")); err != nil {
		t.Fatal(err)
	}
	_, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"mv source alias"}`))
	var failure *toolapi.Failure
	if !errors.As(err, &failure) || failure.Code != toolapi.CommandBlocked {
		t.Fatalf("protected destination alias was not blocked: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(project, "source")); err != nil || string(body) != "keep" {
		t.Fatalf("blocked move changed source: %q %v", body, err)
	}
}

func TestMovePolicyFileToolRejectsNewDestination(t *testing.T) {
	e, project := testExecutor(t)
	if err := os.WriteFile(filepath.Join(project, "source"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("move_file", `{"path":"source","destination":"destination"}`))
	if err != nil || op.Approval().Required {
		t.Fatalf("internal file move required approval: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "destination"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(context.Background(), false); err == nil {
		t.Fatal("stale file move overwrote its destination")
	}
	for name, want := range map[string]string{"source": "keep", "destination": "new"} {
		if body, err := os.ReadFile(filepath.Join(project, name)); err != nil || string(body) != want {
			t.Fatalf("%s changed: %q %v", name, body, err)
		}
	}
}
