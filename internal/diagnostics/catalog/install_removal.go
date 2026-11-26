// Purpose: Own actionable filesystem failure messages for installation removal.
// Pattern: Central diagnostic catalog.
// Usage: The uninstall boundary preserves causes while reporting the affected path.
// Related: internal/commands/install/uninstall.go, internal/diagnostics/fault/error.go.
package catalog

const (
	InstallRemovalAccessDenied Message = "uninstall cannot access %q; check ownership and directory permissions, then rerun meca uninstall (%w)"
	InstallRemovalReadOnly     Message = "uninstall cannot modify %q on a read-only filesystem; restore write access, then rerun meca uninstall (%w)"
	InstallRemovalBusy         Message = "uninstall cannot finish removing %q because it is busy; check for remaining nested mounts, then rerun meca uninstall (%w)"
	InstallRemovalPathFailed   Message = "uninstall could not finish at %q; check the filesystem, then rerun meca uninstall (%w)"
)
