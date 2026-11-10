//go:build tui

// Purpose: Verify tui executable help and server refusal without initializing user state.
// Pattern: Isolated executable boundary regression test.
// Usage: GOFLAGS=-tags=tui go test ./cmd/meca -run TestTUIExecutable.
// Related: cmd/meca/help.go, cmd/meca/server_tui.go, cmd/meca/executable_test.go.
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

func TestTUIExecutable(t *testing.T) {
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
	if err != nil || !strings.Contains(help, "M.E.CA (tui)") || !strings.Contains(help, "--print") || !strings.Contains(help, "upgrade [--tui|--full]") || strings.Contains(help, "--server") {
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
