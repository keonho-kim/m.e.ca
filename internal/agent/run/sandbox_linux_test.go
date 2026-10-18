//go:build linux

// Purpose: Dispatch the private shell entry before running Linux integration fixtures.
// Pattern: Native test process entry.
// Usage: go test ./internal/agent/run.
// Related: internal/tools/local/process/sandbox/linux/child_linux.go.
package run

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
