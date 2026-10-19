//go:build linux || darwin

// Purpose: Verify commands start in the selected project independently of inherited directory state.
// Pattern: Native process contract tests.
// Usage: go test ./internal/tools/local/process -run TestSelectedProjectDirectory.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/process/sandbox/environment.go.
package process

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/tools/local/filesystem"
)

func TestSelectedProjectDirectory(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	canonical, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "selected.txt"), []byte("selected-project"), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := filesystem.New(project, home, filepath.Join(home, ".v"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PWD", "/an/unrelated/project")
	requireTestBoundary(t, b)
	output, err := Run(context.Background(), b, `pwd -P; printf '%s\n' "$PWD"; cat selected.txt`)
	if err != nil || output.ExitCode != 0 || strings.TrimSpace(output.Stdout) != fmt.Sprintf("%s\n%s\nselected-project", canonical, canonical) {
		t.Fatalf("command did not use selected root: %+v %v", output, err)
	}
	if _, err := Run(context.Background(), b, "cd /"); err != nil {
		t.Fatal(err)
	}
	output, err = Run(context.Background(), b, "pwd -P")
	if err != nil || output.ExitCode != 0 || strings.TrimSpace(output.Stdout) != canonical {
		t.Fatalf("previous cd changed the project root: %+v %v", output, err)
	}
}
