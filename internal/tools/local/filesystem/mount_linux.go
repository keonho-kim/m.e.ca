//go:build linux

// Purpose: Detect bind mounts as well as foreign-device descendants at write targets.
// Pattern: Linux filesystem identity Adapter.
// Usage: checkTarget compares kernel mount identities without relying on device numbers alone.
// Related: internal/tools/local/filesystem/target_unix.go.
package filesystem

import (
	"fmt"
	"golang.org/x/sys/unix"
)

func checkMount(root, path string) error {
	rootID, err := mountIdentity(root)
	if err != nil {
		return err
	}
	if root == path {
		return nil
	}
	id, err := mountIdentity(path)
	if err != nil {
		return err
	}
	if rootID != id {
		return fmt.Errorf("write target crosses a mount boundary: %s", path)
	}
	return nil
}
func mountIdentity(path string) (uint64, error) {
	var stat unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_MNT_ID, &stat); err != nil {
		return 0, fmt.Errorf("mount identity unavailable: %w", err)
	}
	if stat.Mask&unix.STATX_MNT_ID == 0 {
		return 0, fmt.Errorf("kernel mount identity unavailable: %s", path)
	}
	return stat.Mnt_id, nil
}
