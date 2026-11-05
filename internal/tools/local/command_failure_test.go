// Purpose: Preserve actual process identity on output failures and avoid duplicate original-output archives.
// Pattern: Projection and filesystem failure regression tests.
// Usage: go test ./internal/tools/local -run 'TestRunningProjection|TestArchiveFailure|TestCommandOriginal'.
// Related: internal/tools/local/command_state.go, internal/tools/local/command_capture.go.
package local

import (
	"encoding/json"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local/process"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunningProjectionFailureKeepsProcessIdentity(t *testing.T) {
	e, _ := testExecutor(t)
	result, err := e.runningCommand(process.State{ID: "cmd_missing", Command: "running command", Status: "running"})
	if err == nil || !result.Started || result.Process == nil || result.Process.CommandID != "cmd_missing" || result.Process.Status != "running" {
		t.Fatalf("lost active process identity on output failure: %+v %v", result, err)
	}
}

func TestRunningObservationDoesNotWaitForArchiveProjection(t *testing.T) {
	e, _ := testExecutor(t)
	c := e.liveOutputs.command("cmd_busy")
	c.mu.Lock()
	defer c.mu.Unlock()
	result, err := e.runningCommand(process.State{ID: "cmd_busy", Command: "command", Status: "finalizing"})
	var body struct {
		Pending bool `json:"output_pending"`
	}
	if err != nil || json.Unmarshal(result.Content, &body) != nil || !body.Pending || result.Process.CommandID != "cmd_busy" {
		t.Fatal("archival hid manageable command status", result, err)
	}
}
func TestArchiveFailureKeepsProcessIdentity(t *testing.T) {
	e, _ := testExecutor(t)
	actual := e.artifacts
	e.artifacts = filepath.Join(actual, "missing")
	defer func() { e.artifacts = actual }()
	op, err := e.Prepare(t.Context(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"printf done"}`))
	if err != nil {
		t.Fatal(err)
	}
	result, err := op.Execute(t.Context(), false)
	if err == nil || !result.Started || result.Process == nil || result.Process.CommandID == "" || result.Process.Status != "exited" {
		t.Fatalf("archive error lost the completed process: %+v %v", result, err)
	}
}
func TestCommandOriginalUsesExistingLogsOnly(t *testing.T) {
	e, _ := testExecutor(t)
	result := commandTool(t, e, "execute_command", map[string]any{"command": "head -c 131072 /dev/zero"})
	original, err := e.CommandCapture(result.Process.CommandID)
	if err != nil || len(original) < 131072 {
		t.Fatal("original was lost", err)
	}
	files, err := os.ReadDir(e.artifacts)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".capture") {
			t.Fatal("stored another full-output original", file.Name())
		}
	}
	if err := e.AcknowledgeCommand(result.Process.CommandID); err != nil {
		t.Fatal(err)
	}
}
