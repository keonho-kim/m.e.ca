// Purpose: Preserve actual deletion facts once without inferring progress from repeated rm success.
// Pattern: Native runtime integration test.
// Usage: go test ./internal/agent/run -run TestShellObservedDeletion.
// Related: internal/agent/run/command_files.go.
package run

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellObservedDeletion(t *testing.T) {
	r, _, root, j, _ := realBatchRunner(t, nil)
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "value"}), nativeCall("execute_command", map[string]any{"command": "rm -f a"}), nativeCall("execute_command", map[string]any{"command": "rm -f a; true"}), "Created and removed a."}}
	r.Generator = g
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.Run(context.Background(), "create a and remove it", approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("status=%s err=%v", result.Status, err)
	}
	for i := len(j.events) - 1; i >= 0; i-- {
		if cp := j.events[i].Checkpoint; cp != nil {
			if !strings.Contains(cp.LastChange, "Deleted") {
				t.Fatalf("latest change is stale: %s", cp.LastChange)
			}
			break
		}
	}
	var deletions int
	for _, e := range j.events {
		if e.Operation == nil {
			continue
		}
		for _, c := range e.Operation.Changes {
			if c.Action == "delete" && c.Path == filepath.Join(root, "a") {
				deletions++
			}
		}
	}
	if deletions != 1 {
		for _, e := range j.events {
			if e.Operation != nil {
				t.Logf("operation: %+v", *e.Operation)
			}
		}
		t.Fatalf("deletions=%d facts=%+v", deletions, result.Facts)
	}
	for _, f := range result.Facts.Files {
		if f.Path == filepath.Join(root, "a") && f.Version == "missing" && !f.Stale {
			return
		}
	}
	t.Fatalf("deletion fact missing: %+v", result.Facts)
}
