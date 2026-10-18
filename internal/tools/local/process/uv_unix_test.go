//go:build linux || darwin

// Purpose: Exercise an explicitly supplied real uv binary installed only in a temporary user's local bin.
// Pattern: Opt-in native tool integration test.
// Usage: Set V_TEST_UV_BINARY and optionally V_TEST_UV_PYTHON_ROOT to an installed managed Python tree.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/process/sandbox/host_environment.go.
package process

import (
	"context"
	"errors"
	"github.com/keonho-kim/v-code/internal/tools/local/filesystem"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUVPackageProject(t *testing.T) {
	binary := os.Getenv("V_TEST_UV_BINARY")
	if binary == "" {
		t.Skip("set V_TEST_UV_BINARY to run against an installed uv")
	}
	home, project := t.TempDir(), t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	private := filepath.Join(home, ".v")
	for _, directory := range []string{bin, private} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if python := os.Getenv("V_TEST_UV_PYTHON_ROOT"); python != "" {
		destination := filepath.Join(home, ".local", "share", "uv", "python", filepath.Base(python))
		if err := os.CopyFS(destination, os.DirFS(python)); err != nil {
			t.Fatal(err)
		}
	}
	source, err := os.Open(binary)
	if err != nil {
		t.Fatal(err)
	}
	target, err := os.OpenFile(filepath.Join(bin, "uv"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		t.Fatal(errors.Join(err, source.Close()))
	}
	_, copyErr := io.Copy(target, source)
	if err := errors.Join(copyErr, source.Close(), target.Close()); err != nil {
		t.Fatal(err)
	}
	// The fixture supplies an explicit user environment and trusted cache exception.
	cache := filepath.Join(home, ".cache", "uv")
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	boundary, err := filesystem.New(project, home, private, []string{cache})
	if err != nil {
		t.Fatal(err)
	}
	requireTestBoundary(t, boundary)
	// Ensure the test does not accidentally use a globally installed uv.
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	output, err := Run(ctx, boundary, "command -v uv && uv --version && uv init --package --offline --vcs none example")
	if err != nil || output.ExitCode != 0 {
		t.Fatalf("%+v %v", output, err)
	}
	if !strings.Contains(output.Stdout, "/.local/bin/uv") {
		t.Fatalf("wrong executable: %s", output.Stdout)
	}
	content, err := os.ReadFile(filepath.Join(project, "example", "pyproject.toml"))
	if err != nil || !strings.Contains(string(content), "[build-system]") {
		t.Fatalf("package project not created: %s %v", content, err)
	}
	t.Log(output.Stdout, output.Stderr)
}
