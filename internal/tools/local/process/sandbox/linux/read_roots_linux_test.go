//go:build linux

// Purpose: Select needed home/tool mounts while omitting unrelated runtime and control submounts.
// Pattern: Projection policy tests.
// Usage: go test ./internal/tools/local/process/sandbox/linux -run TestReadMounts.
// Related: internal/tools/local/process/sandbox/linux/read_roots_linux.go.
package linux

import (
	"slices"
	"testing"
)

func TestReadMounts(t *testing.T) {
	roots := selectReadMounts([]string{"/home/user", "/tools/sdk/bin", "/usr/bin", "/run/control", "/dev/shm"}, []string{"/", "/home", "/home/user", "/tools/sdk", "/usr/lib/wsl", "/run", "/dev/shm"})
	if !slices.Equal(roots, []string{"/home/user", "/tools/sdk"}) {
		t.Fatal(roots)
	}
}
