// Purpose: Read current macOS resident memory with the native proc_info syscall and no CGO.
// Pattern: Native OS adapter with a bounded ABI record.
// Usage: Terminal status sampling calls ResidentBytes at the current UI cadence on macOS.
// Related: internal/processmemory/resident_test.go, internal/tui/session/memory_usage.go.
package processmemory

import (
	"os"
	"syscall"
	"unsafe"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	// ResidentBytes returns the current process's RSS rather than its historical peak.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func ResidentBytes() (uint64, error) {
	// PROC_PIDTASKINFO has six uint64 values followed by twelve int32 counters.
	// Keep the complete 96-byte ABI record; RSS is the second uint64 field.
	var info struct {
		Virtual, Resident uint64
		Times             [4]uint64
		Counters          [12]int32
	}
	const procInfoCallPIDInfo, procPIDTaskInfo = 2, 4
	size := unsafe.Sizeof(info)
	n, _, errno := syscall.Syscall6(syscall.SYS_PROC_INFO, procInfoCallPIDInfo, uintptr(os.Getpid()), procPIDTaskInfo, 0, uintptr(unsafe.Pointer(&info)), size)
	if errno != 0 {
		return 0, errno
	}
	if n != size {
		return 0, faults.Errorf(diagcatalog.ProcessmemoryIncompleteResidentMemorySnapshotOfBytes, n, size)
	}
	return info.Resident, nil
}
