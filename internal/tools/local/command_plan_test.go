// Purpose: Size native command waves from actual remaining owned capacity and benchmark shell preparation.
// Pattern: Real registry admission and fixed-input preparation benchmark.
// Usage: go test ./internal/tools/local -run TestCommandWave -bench BenchmarkRepeatedCommandPreparation -benchmem.
// Related: internal/tools/local/parallel.go, internal/tools/local/command_safety.go.
package local

import (
	"os"
	"strings"
	"testing"
	"time"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local/process"
)

func TestCommandWaveUsesRemainingCapacity(t *testing.T) {
	e, _ := testExecutor(t)
	e.BeginCommands(t.Context(), time.Now().Add(time.Minute))
	defer e.StopCommands()
	for range process.MaxActive - 1 {
		if _, err := e.commands.Start(t.Context(), e.project, "sleep 60", os.Environ(), process.CallIdentity{CallID: "active"}); err != nil {
			t.Fatal(err)
		}
	}
	var calls []toolapi.Call
	for range process.MaxActive {
		calls = append(calls, call("execute_command", `{"command":"printf next"}`))
	}
	plan, err := e.ParallelBatch(t.Context(), policy.Settings{Scope: policy.System}, calls, 16)
	if err != nil || plan.Count != 1 {
		t.Fatal("planned more commands than the remaining capacity", plan, err)
	}
}
func BenchmarkRepeatedCommandPreparation(b *testing.B) {
	home, project := b.TempDir(), b.TempDir()
	e, err := NewExecutor(project, home, home+"/.meca", func(s string) string { return s }, Options{})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := e.Close(); err != nil {
			b.Error(err)
		}
	})
	text := strings.TrimSuffix(strings.Repeat("printf next > result; ", 16), "; ")
	input := call("execute_command", `{"command":"`+text+`"}`)
	b.ReportAllocs()
	for b.Loop() {
		op, err := e.Prepare(b.Context(), policy.Settings{Scope: policy.System}, input)
		if err != nil {
			b.Fatal(err)
		}
		toolapi.Release(op)
	}
}
