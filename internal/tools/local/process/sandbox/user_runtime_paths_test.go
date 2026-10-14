// Purpose: Verify explicit runtime directories cannot redirect to secrets and managed Python receives its real location.
// Pattern: Runtime path contract tests.
// Usage: go test ./internal/tools/local/process/sandbox -run TestUserRuntimePaths.
// Related: internal/tools/local/process/sandbox/user_runtime.go.
package sandbox

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestUserRuntimePathsRejectDirectoryRedirect(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(home, ".local"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(home, filepath.Join(home, ".local", "bin")); err != nil {
		t.Fatal(err)
	}
	if _, err := userRuntime(&Launch{}, home, filepath.Join(home, ".v")); err == nil {
		t.Fatal("home became an exposed runtime")
	}
}

func TestUserRuntimePathsManagedPython(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	python := filepath.Join(home, ".local", "share", "uv", "python")
	if err := os.MkdirAll(python, 0700); err != nil {
		t.Fatal(err)
	}
	launch := Launch{Environment: []string{"PATH=/usr/bin"}}
	paths, err := userRuntime(&launch, home, filepath.Join(home, ".v"))
	if err != nil || !slices.Contains(paths, python) || !slices.Contains(launch.Environment, "UV_PYTHON_INSTALL_DIR="+python) {
		t.Fatalf("%v %v %v", paths, launch.Environment, err)
	}
}
