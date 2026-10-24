//go:build unix

// Purpose: Own captured-log directories and reclaim only captures whose process lease has ended.
// Pattern: Filesystem lifecycle adapter.
// Usage: Executor initialization acquires a capture lease; Close releases it after workers join.
// Related: internal/tools/local/executor.go, internal/tools/local/output.go.
package local

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func newArtifacts() (path string, lease *os.File, err error) {
	base := filepath.Join(os.TempDir(), fmt.Sprintf("meca-output-%d", os.Getuid()))
	if err = os.MkdirAll(base, 0700); err != nil {
		return "", nil, err
	}
	var info unix.Stat_t
	if err = unix.Lstat(base, &info); err != nil {
		return "", nil, err
	}
	if info.Uid != uint32(os.Getuid()) || info.Mode&unix.S_IFMT != unix.S_IFDIR || info.Mode&0077 != 0 {
		return "", nil, fmt.Errorf("captured-output directory must be private and owned by the current user")
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return "", nil, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	guard, err := root.OpenFile("directory.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return "", nil, err
	}
	defer func() { err = errors.Join(err, guard.Close()) }()
	if err = unix.Flock(int(guard.Fd()), unix.LOCK_EX); err != nil {
		return "", nil, err
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "capture-") {
			continue
		}
		child, openErr := root.OpenRoot(entry.Name())
		if errors.Is(openErr, os.ErrNotExist) {
			continue
		}
		if openErr != nil {
			return "", nil, openErr
		}
		owner, openErr := child.OpenFile("owner.lock", os.O_RDWR, 0)
		if closeErr := child.Close(); closeErr != nil {
			if owner != nil {
				closeErr = errors.Join(closeErr, owner.Close())
			}
			return "", nil, closeErr
		}
		if errors.Is(openErr, os.ErrNotExist) {
			if err = root.RemoveAll(entry.Name()); err != nil {
				return "", nil, err
			}
			continue
		}
		if openErr != nil {
			return "", nil, openErr
		}
		lockErr := unix.Flock(int(owner.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if lockErr == nil {
			err = errors.Join(root.RemoveAll(entry.Name()), owner.Close())
			if err != nil {
				return "", nil, err
			}
			continue
		}
		if closeErr := owner.Close(); closeErr != nil {
			return "", nil, closeErr
		}
		if !errors.Is(lockErr, unix.EWOULDBLOCK) && !errors.Is(lockErr, unix.EAGAIN) {
			return "", nil, lockErr
		}
	}
	path, err = os.MkdirTemp(base, "capture-")
	if err != nil {
		return "", nil, err
	}
	lease, err = os.OpenFile(filepath.Join(path, "owner.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err == nil {
		err = unix.Flock(int(lease.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	}
	if err != nil {
		if lease != nil {
			err = errors.Join(err, lease.Close())
		}
		return "", nil, errors.Join(err, os.RemoveAll(path))
	}
	return path, lease, nil
}
