//go:build !tui

// Purpose: Keep optional browser-launch failures separate from server or workspace failures.
// Pattern: Executable regression test with a controlled browser command.
// Usage: go test ./cmd/meca -run TestBrowserLaunchFailureIsWarning.
// Related: cmd/meca/server_ui.go, internal/diagnostics/catalog/events.go.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestBrowserLaunchFailureIsWarning(t *testing.T) {
	binary := testExecutable(t)
	commands := t.TempDir()
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	if err := os.WriteFile(filepath.Join(commands, name), []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	master, terminal, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer terminal.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "--server", "--ui", "--listen", "127.0.0.1:0")
	command.Dir = t.TempDir()
	command.Env = append(os.Environ(), "HOME="+t.TempDir(), "PATH="+commands+string(os.PathListSeparator)+os.Getenv("PATH"))
	command.Stdout = terminal
	output, err := command.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = command.Process.Kill(); _ = command.Wait() }()
	var found struct {
		Level   string `json:"level"`
		Message string `json:"msg"`
	}
	for scanner := bufio.NewScanner(output); scanner.Scan(); {
		if json.Unmarshal(scanner.Bytes(), &found) == nil && (found.Message == "ui_operation_failed" || found.Message == "browser_launch_failed") {
			break
		}
	}
	_ = command.Process.Signal(syscall.SIGTERM)
	if err := command.Wait(); err != nil || ctx.Err() != nil {
		t.Fatal("server did not stay usable after browser launch failure", err, ctx.Err())
	}
	if found.Message != "browser_launch_failed" || found.Level != "WARN" {
		t.Fatalf("optional browser launch reported %s at %s", found.Message, found.Level)
	}
}
