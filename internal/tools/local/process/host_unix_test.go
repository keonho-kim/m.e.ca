//go:build linux || darwin

// Purpose: Verify host resolver visibility and distinguish command startup failures from execution.
// Pattern: Native process contract tests.
// Usage: go test ./internal/tools/local/process -run TestHost.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/process/directory_unix_test.go.
package process

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestHostResolverConfiguration(t *testing.T) {
	content, err := os.ReadFile("/etc/resolv.conf")
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("host has no resolv.conf")
	}
	if err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), t.TempDir(), "cat /etc/resolv.conf", os.Environ())
	if err != nil || result.ExitCode != 0 || result.Stdout != string(content) {
		t.Fatalf("host resolver configuration changed: %+v %v", result, err)
	}
}

func TestHostStartFailure(t *testing.T) {
	for _, directory := range []string{"", "relative", filepath.Join(t.TempDir(), "missing")} {
		result, err := Run(context.Background(), directory, "printf should-not-run", os.Environ())
		if err == nil || result.Started || result.Status != "execution_failed" || result.ExitCode != -1 || result.Stdout != "" {
			t.Fatalf("startup failure lost: %+v %v", result, err)
		}
	}
}
