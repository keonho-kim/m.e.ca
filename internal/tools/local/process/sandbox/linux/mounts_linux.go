//go:build linux

// Purpose: Construct a private filesystem view without exposing host metadata or V state.
// Pattern: Linux mount namespace adapter.
// Usage: The internal child calls enterRoot before applying Landlock and executing the shell.
// Related: internal/tools/local/process/sandbox/linux/child_linux.go, internal/tools/local/process/sandbox/linux/request_linux.go.
package linux

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

var devices = []string{"/dev/null", "/dev/zero", "/dev/random", "/dev/urandom"}

type binding struct {
	path, source              string
	readOnly, device, resolve bool
}

func enterRoot(r Request) error {
	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("isolate mount propagation: %w", err)
	}
	mountTable, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return err
	}
	points, readErr := mountPoints(mountTable)
	if err := errors.Join(readErr, mountTable.Close()); err != nil {
		return err
	}
	budget := 8192
	root := filepath.Join(r.Scratch, "root")
	if err := os.Mkdir(root, 0700); err != nil {
		return err
	}
	if err := unix.Mount("tmpfs", root, "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, "mode=0755,size=16m"); err != nil {
		return err
	}
	for _, path := range r.Runtime {
		if err := bindRuntime(root, path, points, &budget); err != nil {
			return fmt.Errorf("mount runtime %s: %w", path, err)
		}
	}
	for _, path := range []string{r.Project, r.Scratch} {
		if err := bind(root, binding{path: path}); err != nil {
			return fmt.Errorf("mount working directory %s: %w", path, err)
		}
	}
	// User directories must remain canonical at execution and read-only even within the project.
	for _, path := range r.UserRuntime {
		if err := bind(root, binding{path: path, readOnly: true}); err != nil {
			return fmt.Errorf("mount user runtime %s: %w", path, err)
		}
	}
	for _, visible := range append(append([]string{}, r.Runtime...), r.Project) {
		if within(visible, r.Private) {
			if err := hidePrivate(root, r.Private); err != nil {
				return err
			}
			break
		}
	}
	for _, path := range devices {
		if err := bind(root, binding{path: path, device: true, resolve: true}); err != nil {
			return err
		}
	}
	proc := filepath.Join(root, "proc")
	if err := os.MkdirAll(proc, 0755); err != nil {
		return err
	}
	if err := unix.Mount("proc", proc, "proc", unix.MS_RDONLY|unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
		return fmt.Errorf("mount private proc filesystem (host proc masking may prohibit this): %w", err)
	}
	for name, target := range map[string]string{"fd": "/proc/self/fd", "stdin": "/proc/self/fd/0", "stdout": "/proc/self/fd/1", "stderr": "/proc/self/fd/2"} {
		if err := os.Symlink(target, filepath.Join(root, "dev", name)); err != nil {
			return err
		}
	}
	old := filepath.Join(root, ".old-root")
	if err := os.Mkdir(old, 0700); err != nil {
		return err
	}
	if err := unix.PivotRoot(root, old); err != nil {
		return fmt.Errorf("enter isolated root: %w", err)
	}
	if err := unix.Chdir("/"); err != nil {
		return err
	}
	if err := unix.Unmount("/.old-root", unix.MNT_DETACH); err != nil {
		return err
	}
	if err := os.Remove("/.old-root"); err != nil {
		return err
	}
	if err := unix.MountSetattr(unix.AT_FDCWD, "/", 0, &unix.MountAttr{Attr_set: unix.MOUNT_ATTR_RDONLY}); err != nil {
		return err
	}
	return unix.Chdir(r.Project)
}

func bind(root string, mount binding) (err error) {
	// Pin the source inode before constructing the new root. Do not recursively
	// import nested mounts, which could expose host sockets or writable resources.
	resolved := mount.path
	if mount.source != "" {
		resolved = mount.source
	}
	if mount.resolve {
		resolved, err = filepath.EvalSymlinks(mount.path)
		if err != nil {
			return err
		}
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, resolved, &unix.OpenHow{Flags: unix.O_PATH | unix.O_CLOEXEC, Resolve: unix.RESOLVE_NO_SYMLINKS})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, unix.Close(fd)) }()
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return err
	}
	target := filepath.Join(root, mount.path)
	if stat.Mode&unix.S_IFMT == unix.S_IFDIR {
		if err := os.MkdirAll(target, 0755); err != nil {
			return err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
	}
	if err := unix.Mount(fmt.Sprintf("/proc/self/fd/%d", fd), target, "", unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("bind %s: %w", mount.path, err)
	}
	flags := uint64(unix.MOUNT_ATTR_NOSUID)
	if !mount.device {
		flags |= unix.MOUNT_ATTR_NODEV
	}
	if mount.readOnly {
		flags |= unix.MOUNT_ATTR_RDONLY
	}
	if err := unix.MountSetattr(unix.AT_FDCWD, target, 0, &unix.MountAttr{Attr_set: flags}); err != nil {
		return fmt.Errorf("restrict mount %s: %w", mount.path, err)
	}
	return nil
}

func hidePrivate(root, private string) (err error) {
	// Refuse a replaced/symlinked target instead of following it into another mount.
	rootFD, err := unix.Open(root, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, unix.Close(rootFD)) }()
	fd, err := unix.Openat2(rootFD, private[1:], &unix.OpenHow{Flags: unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC, Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, unix.Close(fd)) }()
	return unix.Mount("tmpfs", fmt.Sprintf("/proc/self/fd/%d", fd), "tmpfs", unix.MS_RDONLY|unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, "mode=000,size=4k")
}
