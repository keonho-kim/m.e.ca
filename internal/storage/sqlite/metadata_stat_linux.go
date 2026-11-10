//go:build linux

// Purpose: Fingerprint database and WAL identity and changes while excluding read access times.
// Pattern: Linux metadata adapter.
// Usage: The conversation metadata index validates cached projections before returning them.
// Related: internal/storage/sqlite/conversation_metadata.go.
package sqlite

import (
	"fmt"
	"os"
	"syscall"
)

func metadataVersion(info os.FileInfo) string {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprintf("%d/%d/%d/%d/%d", stat.Dev, stat.Ino, stat.Ctim.Nano(), info.ModTime().UnixNano(), info.Size())
	}
	return fmt.Sprintf("%d/%d", info.ModTime().UnixNano(), info.Size())
}
