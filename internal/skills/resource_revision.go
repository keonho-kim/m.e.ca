// Purpose: Bind a resource page sequence to its pinned file identity and masking revision.
// Pattern: Filesystem version value.
// Usage: ResourceFile returns Revision and requires it on continuation pages.
// Related: internal/skills/resource_file.go, internal/skills/resources.go.
package skills

import (
	"crypto/sha256"
	"fmt"
	"os"
	"syscall"
)

func resourceRevision(info os.FileInfo, version int) string {
	var device, inode uint64
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		device, inode = uint64(stat.Dev), stat.Ino
	}
	identity := fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d", device, inode, info.Size(), info.ModTime().UnixNano(), resourceChangedAt(info), info.Mode(), version)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(identity)))
}
