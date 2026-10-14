//go:build linux

// Purpose: Prevent namespace changes, host socket access and indirect kernel access after sandbox setup.
// Pattern: Seccomp syscall filter adapter.
// Usage: The locked child installs this filter immediately before exec; descendants inherit it.
// Related: internal/tools/local/process/sandbox/linux/child_linux.go.
package linux

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

func restrictSyscalls() error {
	arch := uint32(unix.AUDIT_ARCH_AARCH64)
	if runtime.GOARCH == "amd64" {
		arch = unix.AUDIT_ARCH_X86_64
	}
	load := func(offset uint32) unix.SockFilter {
		return unix.SockFilter{Code: unix.BPF_LD | unix.BPF_W | unix.BPF_ABS, K: offset}
	}
	ret := func(value uint32) unix.SockFilter { return unix.SockFilter{Code: unix.BPF_RET | unix.BPF_K, K: value} }
	equal := func(value uint32, yes, no uint8) unix.SockFilter {
		return unix.SockFilter{Code: unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K, K: value, Jt: yes, Jf: no}
	}
	deny := ret(unix.SECCOMP_RET_ERRNO | uint32(unix.EPERM))
	filter := []unix.SockFilter{
		load(4), equal(arch, 1, 0), ret(unix.SECCOMP_RET_KILL_PROCESS), load(0),
		// Reject x32 numbers even on an x86-64 process; they use another syscall ABI.
		{Code: unix.BPF_JMP | unix.BPF_JSET | unix.BPF_K, K: 0x40000000, Jf: 1}, ret(unix.SECCOMP_RET_KILL_PROCESS),
	}
	for _, number := range []uint32{
		unix.SYS_SOCKET, unix.SYS_CONNECT, unix.SYS_BIND,
		unix.SYS_IO_URING_SETUP, unix.SYS_BPF, unix.SYS_PERF_EVENT_OPEN,
		unix.SYS_PTRACE, unix.SYS_PROCESS_VM_READV, unix.SYS_PROCESS_VM_WRITEV,
		unix.SYS_MOUNT, unix.SYS_UMOUNT2, unix.SYS_PIVOT_ROOT, unix.SYS_CHROOT,
		unix.SYS_SETNS, unix.SYS_UNSHARE, unix.SYS_FSOPEN, unix.SYS_FSMOUNT,
		unix.SYS_FSCONFIG, unix.SYS_MOVE_MOUNT, unix.SYS_OPEN_TREE, unix.SYS_MOUNT_SETATTR,
		unix.SYS_OPEN_BY_HANDLE_AT, unix.SYS_NAME_TO_HANDLE_AT,
	} {
		filter = append(filter, equal(number, 0, 1), deny)
	}
	// FIONBIO (asm-generic/ioctls.h) only changes nonblocking mode on an already
	// open descriptor. Rust uses it to drain child pipes; all device ioctls stay denied.
	filter = append(filter, equal(unix.SYS_IOCTL, 0, 4), load(24),
		equal(0x5421, 0, 1), ret(unix.SECCOMP_RET_ALLOW), deny)
	// clone3 hides flags behind a pointer. ENOSYS lets libc use inspectable clone.
	filter = append(filter, equal(unix.SYS_CLONE3, 0, 1), ret(unix.SECCOMP_RET_ERRNO|uint32(unix.ENOSYS)))
	filter = append(filter, equal(unix.SYS_CLONE, 0, 3), load(16),
		unix.SockFilter{Code: unix.BPF_JMP | unix.BPF_JSET | unix.BPF_K, K: unix.CLONE_NEWUSER | unix.CLONE_NEWNS | unix.CLONE_NEWNET | unix.CLONE_NEWPID | unix.CLONE_NEWIPC | unix.CLONE_NEWUTS | unix.CLONE_NEWCGROUP, Jf: 1}, deny,
		ret(unix.SECCOMP_RET_ALLOW))
	program := unix.SockFprog{Len: uint16(len(filter)), Filter: &filter[0]}
	_, _, errno := unix.Syscall(unix.SYS_SECCOMP, unix.SECCOMP_SET_MODE_FILTER, 0, uintptr(unsafe.Pointer(&program)))
	if errno != 0 {
		return fmt.Errorf("seccomp: %w", errno)
	}
	return nil
}
