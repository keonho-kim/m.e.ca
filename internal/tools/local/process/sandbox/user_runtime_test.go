//go:build linux || darwin

// Purpose: Verify home-installed tools are executable without exposing other home files or writable runtimes.
// Pattern: Native sandbox regression test.
// Usage: go test ./internal/tools/local/process/sandbox -run TestUserRuntime.
// Related: internal/tools/local/process/sandbox/user_runtime.go, internal/tools/local/process/sandbox/filesystem_test.go.
package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestUserRuntimeExecutionAndContainment(t *testing.T) {
	requireSandbox(t)
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, ".local", "bin")
	private := filepath.Join(home, ".v")
	for _, path := range []string{bin, private} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, "v-test-uv"), []byte("#!/bin/sh\nprintf user-tool"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(home, "secret"), filepath.Join(private, "secret")} {
		if err := os.WriteFile(path, []byte("hidden"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(home, "secret"), filepath.Join(bin, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		command, want string
		denied        bool
	}{
		{"command -v v-test-uv", filepath.Join(bin, "v-test-uv"), false},
		{"v-test-uv", "user-tool", false},
		{"printf changed > " + strconv.Quote(filepath.Join(bin, "v-test-uv")), "", true},
		{"cat " + strconv.Quote(filepath.Join(bin, "escape")), "", true},
		{"cat " + strconv.Quote(filepath.Join(private, "secret")), "", true},
	} {
		launch, err := Prepare(context.Background(), project, home, private, tc.command)
		if err != nil {
			t.Fatal(err)
		}
		command := exec.Command(launch.Program, launch.Arguments...)
		command.Dir, command.Env, command.ExtraFiles, command.SysProcAttr = launch.Directory, launch.Environment, launch.Files, launch.Attributes
		if launch.Name != "" {
			command.Args[0] = launch.Name
		}
		var output strings.Builder
		command.Stdout, command.Stderr = &output, &output
		err = command.Start()
		if closeErr := launch.CloseInputs(); closeErr != nil {
			t.Fatal(closeErr)
		}
		if err == nil {
			err = command.Wait()
		}
		setupErr := launch.SetupError()
		closeErr := launch.Close()
		if setupErr != nil || closeErr != nil {
			t.Fatalf("setup=%v close=%v", setupErr, closeErr)
		}
		if (err != nil) != tc.denied || !tc.denied && !strings.Contains(output.String(), tc.want) || strings.Contains(output.String(), "hidden") {
			t.Fatalf("%s: %q %v", tc.command, output.String(), err)
		}
	}
}
