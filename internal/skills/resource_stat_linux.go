// Purpose: Include Linux inode change time in reference page revisions.
// Pattern: Native filesystem metadata adapter.
// Usage: resourceRevision detects rewrites even when a caller restores modification time.
// Related: internal/skills/resource_revision.go, internal/skills/resource_stat_darwin.go.
package skills

import (
	"os"
	"syscall"
)

func resourceChangedAt(info os.FileInfo) int64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return stat.Ctim.Nano()
	}
	return 0
}
