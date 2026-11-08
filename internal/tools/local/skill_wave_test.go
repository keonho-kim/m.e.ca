// Purpose: Prevent unbounded parallel skill image results.
// Pattern: Resource ownership regression test.
// Usage: go test ./internal/tools/local.
// Related: internal/tools/local/parallel.go.
package local

import (
	"encoding/json"
	"path/filepath"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func TestSkillImageReadWaveIsBounded(t *testing.T) {
	home := t.TempDir()
	executor, err := NewExecutor(t.TempDir(), home, filepath.Join(home, ".meca"), func(s string) string { return s }, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	executor.SetSkillImageInput(true)
	calls := make([]toolapi.Call, 16)
	for i := range calls {
		calls[i] = toolapi.Call{ID: "call", Name: "read_skill_file", Arguments: json.RawMessage(`{"id":"project:review","path":"assets/image.png"}`)}
	}
	plan, err := executor.ParallelBatch(t.Context(), policy.Settings{Scope: policy.Directory}, calls, 16)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("skill image wave=%d; planner bytes are based on arguments while each decoded result can retain 32 MiB", plan.Count)
	if plan.Count > 2 {
		t.Fatalf("image-capable skill wave admits %d results against a 64 MiB plan budget", plan.Count)
	}
}
