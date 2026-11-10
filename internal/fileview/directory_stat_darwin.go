//go:build darwin

// Purpose: Bind directory snapshots to identity and change time without access-time noise.
// Pattern: macOS filesystem metadata adapter.
// Usage: Directory revision hashing calls directoryIdentity.
// Related: internal/fileview/directory_snapshot.go.
package fileview

import (
	"fmt"
	"os"
	"syscall"
)

func directoryIdentity(info os.FileInfo) string {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprintf("%d/%d/%d", stat.Dev, stat.Ino, stat.Ctimespec.Nano())
	}
	return info.Name()
}
