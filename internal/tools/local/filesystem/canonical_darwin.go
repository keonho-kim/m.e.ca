//go:build darwin

// Purpose: Normalize actual filesystem casing and firmlink aliases before path policy decisions.
// Pattern: macOS filesystem identity Adapter.
// Usage: Canonical applies F_GETPATH to the nearest existing target before appending missing components.
// Related: internal/tools/local/filesystem/paths.go.
package filesystem

import (
	"bytes"
	"errors"
	"unsafe"

	"golang.org/x/sys/unix"
)

func physicalPath(path string) (result string, err error) {
	fd, err := unix.Open(path, unix.O_EVTONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, unix.Close(fd)) }()
	var buffer [4096]byte
	_, _, errno := unix.Syscall(unix.SYS_FCNTL, uintptr(fd), unix.F_GETPATH, uintptr(unsafe.Pointer(&buffer[0])))
	if errno != 0 {
		return "", errno
	}
	end := bytes.IndexByte(buffer[:], 0)
	if end < 0 {
		return "", unix.ENAMETOOLONG
	}
	return string(buffer[:end]), nil
}
