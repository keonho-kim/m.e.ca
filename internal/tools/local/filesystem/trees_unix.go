//go:build linux || darwin

// Purpose: Reject shared writable inodes and foreign filesystems before shell execution.
// Pattern: Filesystem preflight guard.
// Usage: OS launch preparation checks trusted write trees before exposing them writable.
// Related: internal/tools/local/filesystem/boundary.go.
package filesystem

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"syscall"
)

func (b *Boundary) CheckTrees(ctx context.Context) error {
	if err := b.Check(); err != nil {
		return err
	}
	for _, d := range b.writes {
		root, ok := d.info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("filesystem identity unavailable: %s", d.path)
		}
		err := filepath.WalkDir(d.path, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			for _, p := range b.protected {
				if Within(p, path) {
					if entry.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			stat, ok := info.Sys().(*syscall.Stat_t)
			if !ok {
				return fmt.Errorf("filesystem identity unavailable: %s", path)
			}
			if stat.Dev != root.Dev {
				return fmt.Errorf("nested filesystem in write root: %s", path)
			}
			if info.Mode().IsRegular() && stat.Nlink > 1 {
				return fmt.Errorf("shared inode in write root: %s; use an unlinked copy before execution", path)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
