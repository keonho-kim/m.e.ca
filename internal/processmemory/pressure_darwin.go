// Purpose: Observe macOS memory pressure including memory used by external browser processes.
// Pattern: Native read-only system pressure adapter.
// Usage: Browser admission checks Pressure before provisioning a new session.
// Related: internal/tools/browser/admission.go, internal/processmemory/pressure_linux.go.
package processmemory

import "golang.org/x/sys/unix"

func Pressure() (bool, error) {
	level, err := unix.SysctlUint32("kern.memorystatus_vm_pressure_level")
	return level&6 != 0, err // Darwin warning=2, critical=4, normal=1.
}
