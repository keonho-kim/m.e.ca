//go:build linux || darwin

// Purpose: Verify host environment, resolver visibility and command startup failures.
// Pattern: Native process contract tests.
// Usage: go test ./internal/tools/local/process -run TestHost.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/process/run_test.go.
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

func TestHostEnvironmentReachesChild(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "host-tool")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf '%s' \"$MECA_TEST_VALUE\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	environment := []string{"PATH=" + filepath.Dir(executable), "MECA_TEST_VALUE=preserved"}
	result, err := Run(context.Background(), t.TempDir(), "host-tool; exit $?", environment)
	if err != nil || result.ExitCode != 0 || result.Stdout != "preserved" {
		t.Fatalf("child environment lost: %+v %v", result, err)
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
