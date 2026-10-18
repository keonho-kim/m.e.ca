//go:build linux

// Purpose: Dispatch the private shell entry before running Linux integration fixtures.
// Pattern: Native test process entry.
// Usage: go test ./internal/tools/local.
// Related: internal/tools/local/process/sandbox/linux/child_linux.go.
package local

import (
	"fmt"
	"github.com/keonho-kim/v-code/internal/tools/local/process/sandbox/linux"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if handled, err := linux.Child(); handled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(125)
	}
	os.Exit(m.Run())
}
