// Purpose: Sanitize one completed command once and reuse its value in the actual result and metadata.
// Pattern: Owned process and redaction call-count regression.
// Usage: Executor tests run a local printf in a temporary project and join its archive.
// Related: internal/tools/local/commands.go, internal/tools/local/process/completion.go.
package local

import (
	"context"
	"testing"
	"time"
)

func TestCompletionCollectionSanitizesCommandOnce(t *testing.T) {
	e, _ := testExecutor(t)
	e.BeginCommands(context.Background(), time.Now().Add(time.Minute))
	defer e.StopCommands()
	command := "printf checked"
	result := commandTool(t, e, "execute_command", map[string]any{"command": command})
	if result.Process == nil || result.Process.Status != "exited" {
		t.Fatal("local command did not finish", result)
	}
	redactions := 0
	e.redact = func(text string) string {
		if text == command {
			redactions++
			return "sanitized command"
		}
		return text
	}
	completed, err := e.CollectCommands()
	if err != nil || len(completed) != 1 {
		t.Fatal("completion was lost", completed, err)
	}
	if redactions != 1 || completed[0].Command != "sanitized command" || completed[0].Result.Process.Command != completed[0].Command {
		t.Fatal("command was scanned twice or results used different values", redactions, completed)
	}
}

func BenchmarkEmptyCommandCollection(b *testing.B) {
	e := &Executor{}
	b.ReportAllocs()
	for b.Loop() {
		if results, err := e.CollectCommands(); err != nil || len(results) != 0 {
			b.Fatal("empty registry changed", err)
		}
	}
}
