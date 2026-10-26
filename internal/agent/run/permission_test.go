// Purpose: Verify All Mighty bypasses approval callbacks and records the decision source.
// Pattern: Runtime approval contract test.
// Usage: go test ./internal/agent/run -run TestAllMighty.
// Related: internal/agent/run/prepare_execution.go, internal/policy/permission.go.
package run

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func TestAllMightyAutomaticallyApprovesRequiredOperation(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new"}), "Completed."}}
	e, j := &executor{approval: true}, &journal{}
	r := fixture(g, e, j)
	r.Settings.Permission = policy.PermissionAllMighty
	result, err := r.Run(context.Background(), "Write a", func(context.Context, toolapi.Approval) (bool, error) {
		t.Fatal("All Mighty requested user approval")
		return false, nil
	}, emit)
	if err != nil || result.Status != Answered || e.executed != 1 {
		t.Fatalf("result=%+v executed=%d error=%v", result, e.executed, err)
	}
	found := false
	for _, event := range j.events {
		if event.Kind == "approved" {
			found = event.ApprovalSource == "all_mighty"
		}
	}
	if !found {
		t.Fatal("automatic approval source was not recorded")
	}
}

func TestAllMightyApprovesParallelWave(t *testing.T) {
	root := t.TempDir()
	calls := []toolapi.Call{
		batchCall("a", "create_directory", map[string]any{"paths": []string{filepath.Join(root, "a")}}),
		batchCall("b", "create_directory", map[string]any{"paths": []string{filepath.Join(root, "b")}}),
	}
	r, _, _, _, _ := realBatchRunner(t, calls)
	r.Settings.Permission = policy.PermissionAllMighty
	result, err := r.Run(context.Background(), "create both", func(context.Context, toolapi.Approval) (bool, error) {
		t.Fatal("parallel wave requested user approval")
		return false, nil
	}, emit)
	if err != nil || result.Status != Answered {
		t.Fatal(result.Status, err)
	}
	for _, name := range []string{"a", "b"} {
		if info, err := os.Stat(filepath.Join(root, name)); err != nil || !info.IsDir() {
			t.Fatal("approved wave did not create its directory", name, err)
		}
	}
}
