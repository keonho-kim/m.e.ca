//go:build linux || darwin

// Purpose: Protect active state from installation or deletion by another V process.
// Pattern: Owned directory lease.
// Usage: Startup holds a shared lease; install and uninstall acquire exclusive leases.
// Related: internal/config/statefs/root.go, cmd/v/main.go, internal/commands/install/install.go.
package statefs

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func Lease(root *os.Root, exclusive bool) (*os.File, error) {
	file, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	flags := syscall.LOCK_SH | syscall.LOCK_NB
	if exclusive {
		flags = syscall.LOCK_EX | syscall.LOCK_NB
	}
	if err := syscall.Flock(int(file.Fd()), flags); err != nil {
		return nil, errors.Join(fmt.Errorf("another V process is using application state; close V before installing or uninstalling: %w", err), file.Close())
	}
	return file, nil
}
