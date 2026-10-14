//go:build linux

// Purpose: Enter mandatory kernel isolation and replace the internal child with the approved shell.
// Pattern: Private process entry point.
// Usage: cmd/v dispatches ChildName before interactive startup; failures exit without running a command.
// Related: internal/tools/local/process/sandbox/linux/mounts_linux.go, internal/tools/local/process/sandbox/linux/landlock_linux.go.
package linux

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
)

func Child() (handled bool, err error) {
	if len(os.Args) == 0 || os.Args[0] != ChildName {
		return false, nil
	}
	// The launcher must terminate on either success (exec) or failure. Never
	// unlock and return this restricted thread to the application's worker pool.
	runtime.LockOSThread()
	status := os.NewFile(4, "v-sandbox-status")
	unix.CloseOnExec(4)
	defer func() {
		if err != nil {
			message := err.Error()
			if len(message) > 2048 {
				message = message[:2048]
			}
			_, writeErr := status.WriteString(message)
			err = errors.Join(err, writeErr)
		}
		err = errors.Join(err, status.Close())
	}()
	if os.Getpid() != 1 || os.Getppid() != 0 {
		return true, fmt.Errorf("internal shell requires a fresh PID namespace")
	}
	// An initial host user namespace has a full-range map, not this single UID.
	for _, name := range []string{"uid_map", "gid_map"} {
		mapping, readErr := os.ReadFile("/proc/self/" + name)
		if readErr != nil {
			return true, readErr
		}
		fields := strings.Fields(string(mapping))
		if len(fields) != 3 || fields[0] != "0" || fields[2] != "1" {
			return true, fmt.Errorf("internal shell requires a single-user namespace")
		}
	}
	request, err := readRequest()
	if err != nil {
		return true, err
	}
	if err := Probe(); err != nil {
		return true, err
	}
	if err := enterRoot(request); err != nil {
		return true, fmt.Errorf("filesystem namespace: %w", err)
	}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return true, err
	}
	if err := restrictFiles(request); err != nil {
		return true, fmt.Errorf("Landlock: %w", err)
	}
	if err := dropCapabilities(); err != nil {
		return true, err
	}
	if err := unix.CloseRange(3, uint(^uint32(0)), unix.CLOSE_RANGE_CLOEXEC); err != nil {
		return true, err
	}
	if err := restrictSyscalls(); err != nil {
		return true, err
	}
	if _, err := status.WriteString("ready\n"); err != nil {
		return true, err
	}
	return true, unix.Exec("/bin/sh", []string{"sh", "-c", request.Command}, os.Environ())
}

func dropCapabilities() error {
	// Avoid implicit root capability restoration across exec, even inside the user namespace.
	// SECBIT_NOROOT and SECBIT_NOROOT_LOCKED, from linux/securebits.h.
	if err := unix.Prctl(unix.PR_SET_SECUREBITS, 1|2, 0, 0, 0); err != nil {
		return err
	}
	for capability := uintptr(0); ; capability++ {
		err := unix.Prctl(unix.PR_CAPBSET_DROP, capability, 0, 0, 0)
		if errors.Is(err, unix.EINVAL) {
			break
		}
		if err != nil {
			return err
		}
	}
	header := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	data := [2]unix.CapUserData{}
	return unix.Capset(&header, &data[0])
}
