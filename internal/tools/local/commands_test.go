// Purpose: Exercise native command start, non-destructive waiting, result reuse and cancellation.
// Pattern: Owned command tool integration tests.
// Usage: go test -race ./internal/tools/local -run TestOwnedCommand.
// Related: internal/tools/local/commands.go, internal/tools/local/command_archive.go.
package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func commandTool(t *testing.T, e *Executor, name string, args any) toolapi.Result {
	t.Helper()
	body, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call(name, string(body)))
	if err != nil {
		t.Fatal(err)
	}
	result, err := op.Execute(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestOwnedCommandWaitAndCollect(t *testing.T) {
	e, root := testExecutor(t)
	e.commandWait = time.Millisecond
	e.BeginCommands(context.Background(), time.Now().Add(time.Minute))
	defer e.StopCommands()
	result := commandTool(t, e, "execute_command", map[string]any{"command": "printf begin; while [ ! -f release ]; do sleep 0.02; done; printf finished"})
	if result.Failed || result.Process == nil || result.Process.Status != "running" {
		t.Fatalf("running marked failed: %+v", result)
	}
	id := result.Process.CommandID
	var fields map[string]json.RawMessage
	if json.Unmarshal(result.Content, &fields) != nil || fields["ExitCode"] != nil {
		t.Fatal("running command has an exit code")
	}
	wait := commandTool(t, e, "wait_command", map[string]any{"command_id": id, "wait_seconds": 0})
	if wait.Process.Status != "running" {
		t.Fatal("poll cancelled the command")
	}
	if _, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("write_file", `{"path":"unsafe","content":"x"}`)); err == nil {
		t.Fatal("file mutation overlapped active command")
	}
	if err := os.WriteFile(filepath.Join(root, "release"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	final := commandTool(t, e, "wait_command", map[string]any{"command_id": id, "wait_seconds": 5})
	if final.Failed || final.Process.Status != "exited" || final.Process.ExitCode != 0 {
		t.Fatalf("terminal result %+v", final)
	}
	var output struct{ Stdout string }
	if json.Unmarshal(final.Content, &output) != nil || output.Stdout != "beginfinished" {
		t.Fatalf("lost output %s", final.Content)
	}
	again := commandTool(t, e, "wait_command", map[string]any{"command_id": id, "wait_seconds": 0})
	if again.Process.Status != "exited" {
		t.Fatal("collected result became inaccessible")
	}
}

func TestOwnedCommandCancelAndUnknownIDs(t *testing.T) {
	e, _ := testExecutor(t)
	e.commandWait = time.Millisecond
	e.BeginCommands(context.Background(), time.Now().Add(time.Minute))
	defer e.StopCommands()
	started := commandTool(t, e, "execute_command", map[string]any{"command": "sleep 60"})
	id := started.Process.CommandID
	cancelled := commandTool(t, e, "cancel_command", map[string]any{"command_id": id})
	if cancelled.Failed || cancelled.Process.Status != "interrupted" {
		t.Fatalf("cancel failed %+v", cancelled)
	}
	again := commandTool(t, e, "cancel_command", map[string]any{"command_id": id})
	if again.Process.Status != "interrupted" {
		t.Fatal("cancel was not idempotent")
	}
	for _, id := range []string{"1", "../owner.lock", "cmd_other_session"} {
		body, _ := json.Marshal(map[string]any{"command_id": id, "wait_seconds": 0})
		op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("wait_command", string(body)))
		if err == nil {
			_, err = op.Execute(context.Background(), true)
		}
		if err == nil {
			t.Fatalf("accepted external identity %q", id)
		}
	}
}

func TestOwnedCommandSilentOutputIsUnknown(t *testing.T) {
	e, _ := testExecutor(t)
	e.commandWait = time.Millisecond
	e.BeginCommands(context.Background(), time.Now().Add(time.Minute))
	defer e.StopCommands()
	result := commandTool(t, e, "execute_command", map[string]any{"command": "sleep 60"})
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(result.Content, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["last_output_millis"]) != "null" {
		t.Fatalf("silent tool response invented an output timestamp: %s", result.Content)
	}
	runs := e.CommandActivity()
	if len(runs) != 1 || runs[0].LastOutputMillis != nil {
		t.Fatalf("silent live command invented an output timestamp: %+v", runs)
	}
}
