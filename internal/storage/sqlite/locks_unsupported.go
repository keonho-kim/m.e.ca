//go:build !linux && !darwin

// Purpose: Reject state locking outside the supported Unix platforms.
// Pattern: Platform boundary.
// Usage: Used by database or session startup on unsupported targets.
// Related: internal/storage/sqlite/locks_unix.go, cmd/meca/main.go.
package sqlite

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type SessionLease struct{}

func (*Sessions) LeaseServer(string) (*SessionLease, error) {
	return nil, faults.Errorf(diagcatalog.SqliteServerLockingRequiresLinuxOrMacOS)
}

func (*Store) LeaseSession() (*SessionLease, error) {
	return nil, faults.Errorf(diagcatalog.SqliteSessionLockingRequiresLinuxOrMacOS)
}
func (*SessionLease) Close() error {
	return faults.Errorf(diagcatalog.SqliteSessionLockingRequiresLinuxOrMacOS)
}

func openConversationDeletionLock(string) (*SessionLease, error) {
	return nil, faults.Errorf(diagcatalog.SqliteSessionLockingRequiresLinuxOrMacOS)
}
func openConversationReadLock(string) (*SessionLease, error) {
	return nil, faults.Errorf(diagcatalog.SqliteSessionLockingRequiresLinuxOrMacOS)
}
