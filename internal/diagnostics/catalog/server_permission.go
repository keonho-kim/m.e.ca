// Purpose: Own the global permission invariant shared by concurrent server task admission.
// Pattern: Central message catalog.
// Usage: Server settings reject a global policy change until active tasks have finished.
// Related: internal/commands/server/settings.go.
package catalog

const ServerPermissionTaskActive Message = "finish or cancel active tasks before changing the server permission"
