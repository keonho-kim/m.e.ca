//go:build linux || darwin

// Purpose: Reject shared inodes and mounted descendants at dedicated file mutation targets.
// Pattern: Filesystem preflight guard.
// Usage: WriteTarget checks existing targets and parents before returning an os.Root selection.
// Related: internal/tools/local/filesystem/paths.go, internal/tools/local/filesystem/mount_linux.go.
package filesystem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func checkTarget(root, path string) error {
	rootInfo, err := os.Stat(root)
	if err != nil {
		return err
	}
	rootStat, ok := rootInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("filesystem identity unavailable")
	}
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil {
			stat, ok := info.Sys().(*syscall.Stat_t)
			if !ok || stat.Dev != rootStat.Dev {
				return fmt.Errorf("nested filesystem at write target: %s", current)
			}
			if info.Mode().IsRegular() && stat.Nlink > 1 {
				return fmt.Errorf("shared inode at write target: %s", current)
			}
			if err := checkMount(root, current); err != nil {
				return err
			}
		}
		if current == root {
			return nil
		}
	}
}

func (b *Boundary) RootIdentities() (map[string][2]uint64, error) {
	ids := map[string][2]uint64{}
	for _, d := range b.writes {
		stat, ok := d.info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil, fmt.Errorf("filesystem identity unavailable: %s", d.path)
		}
		ids[d.path] = [2]uint64{uint64(stat.Dev), stat.Ino}
	}
	return ids, nil
}
