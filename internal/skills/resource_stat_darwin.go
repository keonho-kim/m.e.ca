// Purpose: Include macOS inode change time in reference page revisions.
// Pattern: Native filesystem metadata adapter.
// Usage: resourceRevision detects rewrites even when a caller restores modification time.
// Related: internal/skills/resource_revision.go, internal/skills/resource_stat_linux.go.
package skills

import (
	"os"
	"syscall"
)

func resourceChangedAt(info os.FileInfo) int64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return stat.Ctimespec.Nano()
	}
	return 0
}
