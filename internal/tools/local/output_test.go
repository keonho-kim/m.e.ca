// Purpose: Verify large sanitized captures remain readable while model-facing output stays small.
// Pattern: Output lifecycle contract tests.
// Usage: go test ./internal/tools/local -run TestCommandOutput.
// Related: internal/tools/local/output.go, internal/tools/local/executor.go.
package local

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/tools/local/process"
	"os"
	"strings"
	"testing"
)

func TestCommandOutputArchiveAndCleanup(t *testing.T) {
	var archive string
	t.Cleanup(func() {
		if _, err := os.Stat(archive); !os.IsNotExist(err) {
			t.Errorf("capture survived executor shutdown: %v", err)
		}
	})
	e, _ := testExecutor(t)
	source := strings.Repeat("한글 출력\n", 10000)
	excerpt, path, err := e.commandOutput(source, "stdout")
	if err != nil || path == "" || len(excerpt) > outputExcerptBytes+150 {
		t.Fatalf("size=%d path=%s err=%v", len(excerpt), path, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != source {
		t.Fatalf("archive differs: %v", err)
	}
	if _, err = e.boundary.ReadTarget(path); err != nil {
		t.Fatalf("capture cannot be read: %v", err)
	}
	archive = path
}

func TestCommandCaptureIsSanitizedAndReReadable(t *testing.T) {
	e, _ := testExecutor(t)
	settings := policy.Settings{Permission: policy.AllowAll, Scope: policy.System}
	op, err := e.Prepare(context.Background(), settings, call("execute_command", `{"command":"i=0; while [ \"$i\" -lt 10000 ]; do printf '%s%s row\\n' provider- secret; i=$((i+1)); done"}`))
	if err != nil {
		t.Fatal(err)
	}
	result, err := op.Execute(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	var output process.Output
	if err = json.Unmarshal(result.Content, &output); err != nil {
		t.Fatal(err)
	}
	if output.StdoutPath == "" || len(output.Stdout) > outputExcerptBytes+150 || len(result.Capture) == 0 {
		t.Fatalf("missing excerpt/capture: %+v", output)
	}
	if strings.Contains(string(result.Capture), "provider-secret") {
		t.Fatal("secret persisted in original capture")
	}
	body, _ := json.Marshal(map[string]string{"command": "sed -n '10,12p' '" + output.StdoutPath + "'"})
	read, err := e.Prepare(context.Background(), settings, call("execute_command", string(body)))
	if err != nil {
		t.Fatal(err)
	}
	observed, err := read.Execute(context.Background(), true)
	if err != nil || !strings.Contains(string(observed.Content), "[redacted] row") {
		t.Fatalf("capture cannot be reread: %s %v", observed.Content, err)
	}
}
