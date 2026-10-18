//go:build linux

// Purpose: Refuse overwriting a move destination atomically on Linux.
// Pattern: Filesystem syscall Adapter.
// Usage: moveTargets calls renameExclusive with pinned parent descriptors.
// Related: internal/tools/local/move_unix.go.
package local

import "golang.org/x/sys/unix"

func renameExclusive(from int, source string, to int, destination string) error {
	return unix.Renameat2(from, source, to, destination, unix.RENAME_NOREPLACE)
}
