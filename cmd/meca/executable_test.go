// Purpose: Build one CLI fixture lazily and remove it after the executable integration tests finish.
// Pattern: Package-owned test resource.
// Usage: CLI integration tests call testExecutable before starting isolated subprocesses.
// Related: cmd/meca/headless_test.go, cmd/meca/terminal_test.go, cmd/meca/install_shell_test.go.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var cliBuild struct {
	once sync.Once
	dir  string
	path string
	err  error
}

func TestMain(m *testing.M) {
	code := m.Run()
	if cliBuild.dir != "" {
		if err := os.RemoveAll(cliBuild.dir); err != nil {
			fmt.Fprintln(os.Stderr, "remove CLI test executable:", err)
			code = 1
		}
	}
	os.Exit(code)
}

func testExecutable(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds and runs the executable")
	}
	cliBuild.once.Do(func() {
		cliBuild.dir, cliBuild.err = os.MkdirTemp("", "meca-cli-test-*")
		if cliBuild.err != nil {
			return
		}
		cliBuild.path = filepath.Join(cliBuild.dir, "meca")
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if out, err := exec.CommandContext(ctx, "go", "build", "-o", cliBuild.path, ".").CombinedOutput(); err != nil {
			cliBuild.err = fmt.Errorf("build CLI fixture: %w\n%s", err, out)
		}
	})
	if cliBuild.err != nil {
		t.Fatal(cliBuild.err)
	}
	return cliBuild.path
}
