//go:build linux || darwin

// Purpose: Verify application directories belong to the current Unix user.
// Pattern: Filesystem trust boundary.
// Usage: Used before opening or tightening private state permissions.
// Related: internal/config/statefs/root.go, internal/config/statefs/owner_unsupported.go.
package statefs

import (
	"os"
	"syscall"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func checkOwner(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return faults.Errorf(diagcatalog.StatefsApplicationStateMustBelongToTheCurrent)
	}
	return nil
}
