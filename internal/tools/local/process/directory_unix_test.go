//go:build linux || darwin

// Purpose: Verify commands start in the selected project independently of inherited directory state.
// Pattern: Native process contract tests.
// Usage: go test ./internal/tools/local/process -run TestSelectedProjectDirectory.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/process/host_tools_test.go.
package process

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectedProjectDirectory(t *testing.T) {
	project := t.TempDir()
	canonical, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "selected.txt"), []byte("selected-project"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PWD", "/an/unrelated/project")
	output, err := Run(context.Background(), project, `pwd -P; printf '%s\n' "$PWD"; cat selected.txt`, os.Environ())
	if err != nil || output.ExitCode != 0 || strings.TrimSpace(output.Stdout) != fmt.Sprintf("%s\n%s\nselected-project", canonical, canonical) {
		t.Fatalf("command did not use selected root: %+v %v", output, err)
	}
	if _, err := Run(context.Background(), project, "cd /", os.Environ()); err != nil {
		t.Fatal(err)
	}
	output, err = Run(context.Background(), project, "pwd -P", os.Environ())
	if err != nil || output.ExitCode != 0 || strings.TrimSpace(output.Stdout) != canonical {
		t.Fatalf("previous cd changed the project root: %+v %v", output, err)
	}
}
