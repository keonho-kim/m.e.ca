// Purpose: Reject installation when the filesystem cannot publish durable credential storage.
// Pattern: Linux syscall fault injection confined to one test-owned OS thread.
// Usage: go test ./internal/commands/install -run TestInstallDetectsUnsupportedCredentialStorage.
// Related: internal/commands/install/install.go, internal/config/credentials/prepare.go.
package install

import (
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func TestInstallDetectsUnsupportedCredentialStorage(t *testing.T) {
	o := options(t, "/bin/sh")
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	// The runtime retires the locked thread when the test goroutine exits.
	// Other tests keep their normal filesystem operations.
	runtime.LockOSThread()
	filters := []unix.SockFilter{
		{Code: unix.BPF_LD | unix.BPF_W | unix.BPF_ABS, K: 0},
		{Code: unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K, K: uint32(unix.SYS_LINKAT), Jf: 1},
		{Code: unix.BPF_RET | unix.BPF_K, K: unix.SECCOMP_RET_ERRNO | uint32(unix.EOPNOTSUPP)},
		{Code: unix.BPF_RET | unix.BPF_K, K: unix.SECCOMP_RET_ALLOW},
	}
	program := unix.SockFprog{Len: uint16(len(filters)), Filter: &filters[0]}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := unix.Prctl(unix.PR_SET_SECCOMP, unix.SECCOMP_MODE_FILTER, uintptr(unsafe.Pointer(&program)), 0, 0); err != nil {
		t.Fatal(err)
	}
	runtime.KeepAlive(filters)
	if result, err := Install(t.Context(), o); err == nil || result.Binary != "" {
		t.Fatal("installation accepted a filesystem that cannot publish encryption keys", err)
	}
}
