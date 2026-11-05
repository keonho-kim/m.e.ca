// Purpose: Read the current process's resident memory from Linux procfs without spawning a process.
// Pattern: Native OS adapter.
// Usage: Terminal status sampling calls ResidentBytes at the current UI cadence on Linux and WSL.
// Related: internal/processmemory/statm.go, internal/tui/session/memory_usage.go.
package processmemory

import (
	"errors"
	"io"
	"os"
)

// ResidentBytes returns the kernel's current RSS counter, excluding child processes.
func ResidentBytes() (uint64, error) {
	file, err := os.Open("/proc/self/statm")
	if err != nil {
		return 0, err
	}
	var snapshot [128]byte
	n, readErr := file.Read(snapshot[:])
	closeErr := file.Close()
	if readErr == io.EOF && n > 0 {
		readErr = nil
	}
	if err := errors.Join(readErr, closeErr); err != nil {
		return 0, err
	}
	return residentPages(snapshot[:n], uint64(os.Getpagesize()))
}
