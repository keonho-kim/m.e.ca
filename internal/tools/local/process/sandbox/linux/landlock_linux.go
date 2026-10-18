//go:build linux

// Purpose: Require and enforce kernel filesystem access rules on the thread that will execute the shell.
// Pattern: Linux Landlock syscall adapter.
// Usage: Probe runs in the parent; restrictFiles runs only in the locked child thread.
// Related: internal/tools/local/process/sandbox/linux/child_linux.go.
package linux

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

const readAccess = unix.LANDLOCK_ACCESS_FS_EXECUTE | unix.LANDLOCK_ACCESS_FS_READ_FILE | unix.LANDLOCK_ACCESS_FS_READ_DIR
const fileAccess = readAccess | unix.LANDLOCK_ACCESS_FS_WRITE_FILE | unix.LANDLOCK_ACCESS_FS_REMOVE_DIR | unix.LANDLOCK_ACCESS_FS_REMOVE_FILE | unix.LANDLOCK_ACCESS_FS_MAKE_CHAR | unix.LANDLOCK_ACCESS_FS_MAKE_DIR | unix.LANDLOCK_ACCESS_FS_MAKE_REG | unix.LANDLOCK_ACCESS_FS_MAKE_SOCK | unix.LANDLOCK_ACCESS_FS_MAKE_FIFO | unix.LANDLOCK_ACCESS_FS_MAKE_BLOCK | unix.LANDLOCK_ACCESS_FS_MAKE_SYM | unix.LANDLOCK_ACCESS_FS_REFER | unix.LANDLOCK_ACCESS_FS_TRUNCATE

func requireABI(abi int) error {
	if abi < 3 {
		return fmt.Errorf("Landlock ABI 3 or newer is required (available: %d)", abi)
	}
	return nil
}

func Probe() error {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return fmt.Errorf("kernel sandbox is unsupported on %s", runtime.GOARCH)
	}
	abi, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	if errno != 0 {
		return fmt.Errorf("Landlock unavailable: %w", errno)
	}
	return requireABI(int(abi))
}

func restrictFiles(request Request) (err error) {
	// Use the complete ABI 3 filesystem contract; never drop unsupported rights.
	access := uint64(fileAccess)
	fd, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, uintptr(unsafe.Pointer(&access)), unsafe.Sizeof(access), 0)
	if errno != 0 {
		return errno
	}
	defer func() { err = errors.Join(err, unix.Close(int(fd))) }()
	if err := allowPath(int(fd), "/", readAccess); err != nil {
		return err
	}
	writes := append(append([]string{}, request.Writable...), request.Scratch)
	for _, path := range writes {
		if err := allowPath(int(fd), path, fileAccess); err != nil {
			return err
		}
	}
	for _, path := range devices {
		access := uint64(unix.LANDLOCK_ACCESS_FS_READ_FILE)
		if path == "/dev/null" || path == "/dev/zero" {
			access |= unix.LANDLOCK_ACCESS_FS_WRITE_FILE
		}
		if err := allowPath(int(fd), path, access); err != nil {
			return err
		}
	}
	_, _, errno = unix.Syscall(unix.SYS_LANDLOCK_RESTRICT_SELF, fd, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func allowPath(ruleset int, path string, access uint64) (err error) {
	fd, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("Landlock path %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, unix.Close(fd)) }()
	rule := unix.LandlockPathBeneathAttr{Allowed_access: access, Parent_fd: int32(fd)}
	_, _, errno := unix.Syscall6(unix.SYS_LANDLOCK_ADD_RULE, uintptr(ruleset), unix.LANDLOCK_RULE_PATH_BENEATH, uintptr(unsafe.Pointer(&rule)), 0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("Landlock rule %s: %w", path, errno)
	}
	return nil
}
