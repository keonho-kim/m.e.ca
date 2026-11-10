//go:build minimum

// Purpose: Verify minimum executable help and server refusal without initializing user state.
// Pattern: Isolated executable boundary regression test.
// Usage: GOFLAGS=-tags=minimum go test ./cmd/meca -run TestMinimumExecutable.
// Related: cmd/meca/help.go, cmd/meca/server_minimum.go, cmd/meca/executable_test.go.
package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMinimumExecutable(t *testing.T) {
	binary := testExecutable(t)
	home, project := t.TempDir(), t.TempDir()
	run := func(args ...string) (string, error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, binary, args...)
		command.Dir = project
		command.Env = append(os.Environ(), "HOME="+home)
		output, err := command.CombinedOutput()
		return string(output), err
	}
	help, err := run("--help")
	if err != nil || !strings.Contains(help, "M.E.CA (minimum)") || !strings.Contains(help, "--print") || strings.Contains(help, "--server") {
		t.Fatal(help, err)
	}
	output, err := run("--server", "--ui")
	if err == nil || !strings.Contains(output, "full") {
		t.Fatal(output, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".meca")); !os.IsNotExist(err) {
		t.Fatal("unavailable server entry initialized application state", err)
	}
}
