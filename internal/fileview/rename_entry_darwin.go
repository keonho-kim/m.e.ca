// Purpose: Move an entry atomically without replacing a destination on macOS.
// Pattern: Platform filesystem adapter.
// Usage: renameEntry supplies open directory descriptors.
// Related: internal/fileview/rename_entry_unix.go.
package fileview

import "golang.org/x/sys/unix"

func renameNoReplace(from int, name string, to int, next string) error {
	return unix.RenameatxNp(from, name, to, next, unix.RENAME_EXCL)
}
