//go:build linux || darwin

// Purpose: Verify search commands use the host shell consistently, including pipelines and redirects.
// Pattern: Process integration regression test.
// Usage: go test ./internal/tools/local -run TestCommandSearchPipeline.
// Related: internal/tools/local/command.go, internal/tools/local/command_policy_test.go.
package local

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandSearchPipeline(t *testing.T) {
	executor, root := testExecutor(t)
	if err := os.WriteFile(filepath.Join(root, "app.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"find . -name 'app.json' -type f 2>/dev/null | head -20"}`))
	if err != nil {
		t.Fatal(err)
	}
	result, err := op.Execute(context.Background(), false)
	if err != nil || result.Failed {
		t.Fatalf("%+v %v", result, err)
	}
	var output struct {
		Stdout, Status string
		ExitCode       int
	}
	if err := json.Unmarshal(result.Content, &output); err != nil {
		t.Fatal(err)
	}
	if output.Status != "exited" || output.ExitCode != 0 || strings.TrimSpace(output.Stdout) != "./app.json" {
		t.Fatalf("unexpected shell output: %+v", output)
	}
}
