//go:build linux || darwin

// Purpose: Verify terminal installation activates a login shell only when needed and releases installation state first.
// Pattern: Executable and PTY integration tests with an isolated home and probe shell.
// Usage: go test ./cmd/meca -run TestInstallShellExecutable.
// Related: cmd/meca/install.go, cmd/meca/install_shell.go, internal/commands/startup/parse.go.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestInstallShellExecutable(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the installation executable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "meca")
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %s %v", out, err)
	}
	for _, tc := range []struct {
		name                       string
		terminal, disabled, inPath bool
	}{
		{name: "terminal", terminal: true},
		{name: "no-shell", terminal: true, disabled: true},
		{name: "already available", terminal: true, inPath: true},
		{name: "nonterminal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			shell := filepath.Join(t.TempDir(), "sh")
			probe := "#!/bin/sh\nset -eu\nprintf 'LOGIN_ARGS:%s\\nLOGIN_PID:%s\\n' \"$*\" \"$$\"\ncommand -v meca\n\"$HOME/.local/bin/meca\" install --no-shell\n"
			if err := os.WriteFile(shell, []byte(probe), 0755); err != nil {
				t.Fatal(err)
			}
			path := "/usr/bin:/bin"
			if tc.inPath {
				path = filepath.Join(home, ".local/bin") + ":" + path
			}
			args := []string{"install"}
			if tc.disabled {
				args = append(args, "--no-shell")
			}
			command := exec.CommandContext(ctx, binary, args...)
			command.Env = append(os.Environ(), "HOME="+home, "SHELL="+shell, "PATH="+path)
			var output []byte
			if tc.terminal {
				master, startErr := pty.Start(command)
				if startErr != nil {
					t.Fatal(startErr)
				}
				defer master.Close()
				output, _ = io.ReadAll(master) // A closed PTY may report EIO after delivering its final output.
				err = command.Wait()
			} else {
				output, err = command.CombinedOutput()
			}
			if err != nil {
				t.Fatalf("install: %v\n%s", err, output)
			}
			active := strings.Contains(string(output), "LOGIN_ARGS:-l")
			if active != (tc.terminal && !tc.disabled && !tc.inPath) {
				t.Fatalf("unexpected shell activation:\n%s", output)
			}
			if active && (!strings.Contains(string(output), fmt.Sprintf("LOGIN_PID:%d", command.Process.Pid)) ||
				!strings.Contains(string(output), filepath.Join(home, ".local/bin/meca"))) {
				t.Fatalf("login shell was spawned separately or could not find meca:\n%s", output)
			}
		})
	}
}
