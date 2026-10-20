// Purpose: Verify moves can use host paths and preserve destination exclusivity.
// Pattern: Filesystem integration test.
// Usage: go test ./internal/tools/local -run TestMoveAcrossRoots.
// Related: internal/tools/local/move_unix.go, internal/tools/local/candidate.go.
package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/v-code/internal/policy"
)

func TestMoveAcrossRoots(t *testing.T) {
	project, home, extra := t.TempDir(), t.TempDir(), t.TempDir()
	e, err := NewExecutor(project, home, filepath.Join(home, ".v"), func(s string) string { return s })
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := e.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := os.WriteFile(filepath.Join(project, "source"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(extra, "destination")
	settings := policy.Settings{Scope: policy.System}
	op, err := e.Prepare(context.Background(), settings, call("move_file", fmt.Sprintf(`{"path":"source","destination":%q}`, dest)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(dest); err != nil || string(data) != "original" {
		t.Fatal("wrong destination", err)
	}
	if _, err := os.Stat(filepath.Join(project, "source")); !os.IsNotExist(err) {
		t.Fatal("source remains")
	}
	if _, err := e.Prepare(context.Background(), settings, call("move_file", `{"path":".","destination":"other"}`)); err == nil {
		t.Fatal("project root moved")
	}
}
