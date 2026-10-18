// Purpose: Exercise routing and execution contracts through real filesystem adapters without live models.
// Pattern: Scripted integration regressions for observed failures.
// Usage: go test ./internal/agent/run -run TestAlignment.
// Related: internal/agent/run/grounding.go, internal/agent/run/batch_wave.go, internal/tools/local/directions.go.
package run

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAlignmentShellCreationFeedbackNamesActualNativeArguments(t *testing.T) {
	r, _, root, j, _ := realBatchRunner(t, nil)
	g := &generator{replies: []string{nativeCall("execute_command", map[string]any{"command": "mkdir -p src/app"}), nativeCall("create_directory", map[string]any{"paths": []string{"src/app"}}), "Created src/app."}}
	r.Generator = g
	result, err := r.Run(context.Background(), "create src/app", approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("%+v %v", result, err)
	}
	if info, err := os.Stat(filepath.Join(root, "src/app")); err != nil || !info.IsDir() {
		t.Fatal("native recovery did not create directory")
	}
	for _, m := range j.messages {
		if m.Role == "tool" && m.ToolName == "execute_command" {
			if !strings.Contains(m.Content, "paths: an array") || !strings.Contains(m.Content, "Nothing from the rejected shell command ran") {
				t.Fatalf("unclear routing: %s", m.Content)
			}
			return
		}
	}
	t.Fatal("missing structured routing feedback")
}
